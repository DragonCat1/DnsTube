package dns

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/dnstube/dnstube/internal/store"
	mdns "github.com/miekg/dns"
)

type compiledRule struct {
	re       *regexp.Regexp
	ruleID   int32 // forward_rules.id，用于统计命中次数
	targetID int32
	mode     string
	priority int
}

type instanceSnap struct {
	instance  store.DNSInstance
	records   []store.DNSRecord
	rules     []compiledRule          // 仅通配符/正则/关键字规则
	domainMap map[string]compiledRule // 纯域名规则
	groups    map[int32][]store.UpstreamServer
}

func fqdnEqual(a, b string) bool {
	return strings.EqualFold(mdns.Fqdn(a), mdns.Fqdn(b))
}

func tryStaticAnswer(req *mdns.Msg, recs []store.DNSRecord) (*mdns.Msg, bool) {
	if len(req.Question) == 0 {
		return nil, false
	}
	q := req.Question[0]
	qname := mdns.Fqdn(q.Name)
	qtype := q.Qtype

	m := new(mdns.Msg)
	m.SetReply(req)
	m.Authoritative = true
	m.Rcode = mdns.RcodeNameError

	var matched bool
	for _, r := range recs {
		if !fqdnEqual(r.Name, qname) {
			continue
		}
		rt := typeFromRecord(r.Rtype)
		if rt == mdns.TypeNone || rt != qtype {
			continue
		}
		rr, err := mdns.NewRR(fmt.Sprintf("%s %d %s %s", qname, r.TTL, strings.ToUpper(r.Rtype), r.Content))
		if err != nil {
			continue
		}
		m.Answer = append(m.Answer, rr)
		matched = true
		m.Rcode = mdns.RcodeSuccess
	}
	if !matched {
		return nil, false
	}
	return m, true
}

// ruleIsBetter 返回 true 表示 a 优先于 b（priority 小优先，其次 targetID、ruleID）。
func ruleIsBetter(a, b compiledRule) bool {
	if a.priority != b.priority {
		return a.priority < b.priority
	}
	if a.targetID != b.targetID {
		return a.targetID < b.targetID
	}
	return a.ruleID < b.ruleID
}

func pickForwardRule(qname string, rules []compiledRule, domainMap map[string]compiledRule) *compiledRule {
	name := strings.ToLower(strings.TrimSuffix(qname, "."))

	// 1. 域名后缀逐级查 map
	var domainHit *compiledRule
	for lookup := name; lookup != ""; {
		if r, ok := domainMap[lookup]; ok {
			if domainHit == nil || ruleIsBetter(r, *domainHit) {
				cp := r
				domainHit = &cp
			}
		}
		idx := strings.Index(lookup, ".")
		if idx < 0 {
			break
		}
		lookup = lookup[idx+1:]
	}

	// 2. 线性扫正则规则（已按优先级排好序，首命中即最佳）
	var regexHit *compiledRule
	for i := range rules {
		if rules[i].re != nil && rules[i].re.MatchString(name) {
			regexHit = &rules[i]
			break
		}
	}

	// 3. 取优先级更高的
	if domainHit == nil {
		return regexHit
	}
	if regexHit == nil {
		return domainHit
	}
	if ruleIsBetter(*domainHit, *regexHit) {
		return domainHit
	}
	return regexHit
}

func (e *Engine) handleQuery(ctx context.Context, inst *instanceSnap, remote net.Addr, req *mdns.Msg, start time.Time) *mdns.Msg {
	if len(req.Question) == 0 {
		m := new(mdns.Msg)
		m.SetRcode(req, mdns.RcodeFormatError)
		return m
	}
	qctx, cancel := context.WithTimeout(ctx, e.queryTotalTimeout)
	defer cancel()

	q := req.Question[0]
	qname := mdns.Fqdn(q.Name)
	clientIP := "0.0.0.0"
	if u, ok := remote.(*net.UDPAddr); ok {
		clientIP = u.IP.String()
	}

	key := cacheKey(inst.instance.ID, qname, q.Qtype)
	if msg, ok := e.cache.get(key); ok {
		rc := mdns.RcodeNameError
		if msg != nil {
			rc = msg.Rcode
		}
		e.logQuery(qctx, inst.instance.ID, clientIP, qname, qtypeString(q.Qtype), mdns.RcodeToString[rc], true, false, nil, nil, int(time.Since(start).Milliseconds()), nil, msg, nil)
		if msg == nil {
			return nil
		}
		// 缓存命中返回的是历史应答副本，其 Msg.Id 仍是写入缓存时的值；必须对齐到当前请求，
		// 否则 dig 等客户端会报 ID mismatch。不能用 SetReply(req)：它会强制 Rcode=Success，破坏 NXDOMAIN 等缓存。
		msg.Id = req.Id
		return msg
	}

	if ans, ok := tryStaticAnswer(req, inst.records); ok {
		e.cache.set(key, ans, minTTL(ans))
		e.logQuery(qctx, inst.instance.ID, clientIP, qname, qtypeString(q.Qtype), mdns.RcodeToString[ans.Rcode], false, false, nil, nil, int(time.Since(start).Milliseconds()), nil, ans, nil)
		return ans
	}

	reqCopy := req.Copy()
	// 禁止调用 SetQuestion：miekg/dns 的 SetQuestion 会执行 Id = Id()，覆盖客户端事务 ID，
	// 上游应答将带随机 ID，dig 会报 ID mismatch。仅就地更新 Question，保留 req 的 Id 与其它头字段。
	if len(reqCopy.Question) > 0 {
		reqCopy.Question[0].Name = qname
		reqCopy.Question[0].Qtype = q.Qtype
		reqCopy.Question[0].Qclass = q.Qclass
	}

	var resp *mdns.Msg
	var upAddr string
	var upMs int
	var fwdErr error
	var forwarded bool
	var fwdGroupID *int32

	if rule := pickForwardRule(qname, inst.rules, inst.domainMap); rule != nil {
		rid := rule.ruleID
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = e.store.IncrementForwardRuleHitCount(ctx, rid)
		}()
		gid := rule.targetID
		fwdGroupID = &gid
		servers := inst.groups[rule.targetID]
		if rule.mode == "parallel" {
			resp, upAddr, upMs, fwdErr = ParallelForward(e.exchangeClient, qctx, servers, reqCopy)
		} else {
			resp, upAddr, upMs, fwdErr = SequentialForward(e.exchangeClient, qctx, servers, reqCopy)
		}
		forwarded = true
	} else if inst.instance.DefaultUpstreamGroupID != nil {
		fwdGroupID = inst.instance.DefaultUpstreamGroupID
		servers := inst.groups[*inst.instance.DefaultUpstreamGroupID]
		resp, upAddr, upMs, fwdErr = ParallelForward(e.exchangeClient, qctx, servers, reqCopy)
		forwarded = true
	} else {
		m := new(mdns.Msg)
		m.SetReply(req)
		m.Rcode = mdns.RcodeServerFailure
		errs := "no upstream configured"
		e.logQuery(qctx, inst.instance.ID, clientIP, qname, qtypeString(q.Qtype), "SERVFAIL", false, false, nil, nil, int(time.Since(start).Milliseconds()), &errs, nil, nil)
		return m
	}

	if fwdErr != nil || resp == nil {
		m := new(mdns.Msg)
		m.SetReply(req)
		m.Rcode = mdns.RcodeServerFailure
		es := fwdErr.Error()
		upPtr := upAddr
		var ums *int
		if upMs > 0 {
			ums = &upMs
		}
		e.logQuery(qctx, inst.instance.ID, clientIP, qname, qtypeString(q.Qtype), "SERVFAIL", false, forwarded, &upPtr, ums, int(time.Since(start).Milliseconds()), &es, nil, fwdGroupID)
		return m
	}

	resp.Id = req.Id
	// 上游明确返回 SERVFAIL 时不写缓存，避免将瞬时上游异常放大到后续请求。
	if resp.Rcode != mdns.RcodeServerFailure {
		e.cache.set(key, resp, minTTL(resp))
	}
	upPtr := upAddr
	ums := upMs
	e.logQuery(qctx, inst.instance.ID, clientIP, qname, qtypeString(q.Qtype), mdns.RcodeToString[resp.Rcode], false, forwarded, &upPtr, &ums, int(time.Since(start).Milliseconds()), nil, resp, fwdGroupID)
	return resp
}

func minTTL(msg *mdns.Msg) time.Duration {
	var t uint32 = 300
	if len(msg.Answer) > 0 {
		t = msg.Answer[0].Header().Ttl
		for _, a := range msg.Answer {
			if a.Header().Ttl < t {
				t = a.Header().Ttl
			}
		}
	}
	if t == 0 {
		return 5 * time.Minute
	}
	return time.Duration(t) * time.Second
}

func (e *Engine) logQuery(ctx context.Context, instanceID int32, clientIP, qname, qtype, rcode string, cacheHit, forwarded bool, up *string, upMs *int, totalMs int, errMsg *string, respMsg *mdns.Msg, forwardGroupID *int32) {
	rs := summarizeForQueryLog(errMsg, respMsg)
	tm := totalMs
	q := store.QueryLogInsert{
		InstanceID:    instanceID,
		ClientIP:      clientIP,
		Qname:         qname,
		Qtype:         qtype,
		ResponseCode:  rcode,
		CacheHit:      cacheHit,
		Forwarded:     forwarded,
		TotalMs:       &tm,
		ResultSummary: rs,
	}
	if up != nil {
		s := *up
		q.UpstreamAddr = &s
	}
	if upMs != nil {
		ms := *upMs
		q.UpstreamMs = &ms
	}
	if errMsg != nil {
		s := *errMsg
		q.ErrorMessage = &s
	}
	if forwardGroupID != nil {
		g := *forwardGroupID
		q.ForwardUpstreamGroupID = &g
	}
	e.enqueueQueryLog(ctx, q)
}
