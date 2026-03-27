package dns

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dnstube/dnstube/internal/store"
	mdns "github.com/miekg/dns"
)

// Engine runs UDP DNS listeners and resolves using DB-backed snapshots.
type Engine struct {
	store        *store.Store
	cache        *queryCache
	log          *slog.Logger
	mu           sync.RWMutex
	snaps        map[int32]*instanceSnap
	cancel       context.CancelFunc
	rootCtx      context.Context
	wg           sync.WaitGroup
	listenerMu   sync.Mutex
	listeners    map[int32]context.CancelFunc
	listenerBind map[int32]string
	// listenerErr 记录 UDP 绑定失败（如端口占用）；与 DB 中 paused 等正交，供 API 展示。
	listenerErr map[int32]string

	queryLogMu       sync.Mutex
	queryLogBuf      []store.QueryLogInsert
	queryLogWG       sync.WaitGroup
	queryLogOverflow atomic.Uint64

	// dnsClient 供上游转发复用，避免每次 Exchange 分配新 Client。
	dnsClient *mdns.Client

	queryTotalTimeout    time.Duration
	maxConcurrentQueries int
	udpHandlersWG        sync.WaitGroup
}

func NewEngine(st *store.Store, log *slog.Logger, upstreamTimeout, queryTotalTimeout time.Duration, maxConcurrentQueries int) *Engine {
	if log == nil {
		log = slog.Default()
	}
	if upstreamTimeout <= 0 {
		upstreamTimeout = DefaultUDPUpstreamExchangeTimeout
	}
	if queryTotalTimeout <= upstreamTimeout {
		queryTotalTimeout = upstreamTimeout + time.Second
	}
	if maxConcurrentQueries <= 0 {
		maxConcurrentQueries = 256
	}
	return &Engine{
		store:                st,
		cache:                newQueryCache(10000, 5*time.Minute),
		log:                  log,
		snaps:                make(map[int32]*instanceSnap),
		listeners:            make(map[int32]context.CancelFunc),
		listenerBind:         make(map[int32]string),
		listenerErr:          make(map[int32]string),
		dnsClient:            &mdns.Client{Net: "udp", Timeout: upstreamTimeout},
		queryTotalTimeout:    queryTotalTimeout,
		maxConcurrentQueries: maxConcurrentQueries,
	}
}

func (e *Engine) Reload(ctx context.Context) error {
	instances, err := e.store.ListInstances(ctx)
	if err != nil {
		return err
	}
	next := make(map[int32]*instanceSnap)
	for _, inst := range instances {
		snap, err := e.loadInstance(ctx, inst)
		if err != nil {
			e.log.Error("load instance", "id", inst.ID, "err", err)
			continue
		}
		next[inst.ID] = snap
	}
	e.mu.Lock()
	e.snaps = next
	e.mu.Unlock()
	return nil
}

func (e *Engine) loadInstance(ctx context.Context, inst store.DNSInstance) (*instanceSnap, error) {
	recs, err := e.store.ListRecordsByInstance(ctx, inst.ID)
	if err != nil {
		return nil, err
	}
	rulesDB, err := e.store.ListForwardRules(ctx, inst.ID)
	if err != nil {
		return nil, err
	}
	var rules []compiledRule

	// domainEntries 收集纯域名规则，稍后构建 map。
	type domainEntry struct {
		domain string
		rule   compiledRule
	}
	var domainEntries []domainEntry

	for _, r := range rulesDB {
		if r.Disabled {
			continue
		}
		pats, domains, compErr := compilePatternsForEngine(ctx, r)
		if compErr != nil {
			e.log.Warn("skip forward rule", "id", r.ID, "err", compErr)
			continue
		}
		if len(pats) == 0 && len(domains) == 0 {
			e.log.Warn("skip forward rule: no patterns", "id", r.ID)
			continue
		}
		cr := compiledRule{ruleID: r.ID, targetID: r.TargetGroupID, mode: r.Mode, priority: r.Priority}
		for _, re := range pats {
			if re == nil {
				continue
			}
			rc := cr
			rc.re = re
			rules = append(rules, rc)
		}
		for _, d := range domains {
			domainEntries = append(domainEntries, domainEntry{domain: d, rule: cr})
		}
	}
	// 与 pickForwardRule 原排序键一致；稳定排序以保留同优先级、同组内的加载顺序。
	sort.SliceStable(rules, func(i, j int) bool {
		if rules[i].priority != rules[j].priority {
			return rules[i].priority < rules[j].priority
		}
		if rules[i].targetID != rules[j].targetID {
			return rules[i].targetID < rules[j].targetID
		}
		return rules[i].ruleID < rules[j].ruleID
	})
	// 构建 domainMap：同域名保留优先级最高的规则。
	domainMap := make(map[string]compiledRule, len(domainEntries))
	for _, de := range domainEntries {
		if existing, ok := domainMap[de.domain]; ok {
			if ruleIsBetter(de.rule, existing) {
				domainMap[de.domain] = de.rule
			}
		} else {
			domainMap[de.domain] = de.rule
		}
	}
	groups := make(map[int32][]store.UpstreamServer)
	seen := map[int32]struct{}{}
	for _, r := range rulesDB {
		if r.Disabled {
			continue
		}
		seen[r.TargetGroupID] = struct{}{}
	}
	if inst.DefaultUpstreamGroupID != nil {
		seen[*inst.DefaultUpstreamGroupID] = struct{}{}
	}
	for gid := range seen {
		srv, err := e.store.ListUpstreamServers(ctx, gid)
		if err != nil {
			return nil, err
		}
		groups[gid] = srv
	}
	return &instanceSnap{instance: inst, records: recs, rules: rules, domainMap: domainMap, groups: groups}, nil
}

// RefreshAllURLPatterns 按 URL 拉取所有启用中的转发规则并写库；失败则跳过该条并保留上次成功缓存。
func (e *Engine) RefreshAllURLPatterns(ctx context.Context) error {
	rules, err := e.store.ListForwardRulesURLSource(ctx)
	if err != nil {
		return err
	}
	for _, r := range rules {
		body, count, lines, err := FetchURLPattern(ctx, r)
		if err != nil {
			e.log.Warn("scheduled url pattern refresh failed", "rule_id", r.ID, "err", err)
			continue
		}
		enc, mErr := EncodedResolvedEntries(lines)
		if mErr != nil {
			e.log.Error("marshal resolved pattern entries", "rule_id", r.ID, "err", mErr)
			continue
		}
		if err := e.store.UpdateForwardRulePatternFetch(ctx, r.ID, body, count, time.Now().UTC(), enc); err != nil {
			e.log.Error("update forward rule pattern fetch", "rule_id", r.ID, "err", err)
			continue
		}
	}
	return e.Reload(ctx)
}

// Start launches periodic reload, syncs UDP listeners, until ctx is cancelled.
func (e *Engine) Start(ctx context.Context) error {
	e.rootCtx = ctx
	if err := e.Reload(ctx); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	e.cancel = cancel

	e.queryLogWG.Add(1)
	go e.queryLogFlushLoop(ctx)

	_ = e.syncListeners(ctx)

	tick := time.NewTicker(5 * time.Second)
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				if err := e.Reload(context.Background()); err != nil {
					e.log.Error("reload dns config", "err", err)
				}
				_ = e.syncListeners(ctx)
			}
		}
	}()
	return nil
}

func (e *Engine) syncListeners(ctx context.Context) error {
	type wantT struct {
		bind   string
		paused bool
	}
	e.mu.RLock()
	want := make(map[int32]wantT)
	for id, snap := range e.snaps {
		want[id] = wantT{
			bind:   fmt.Sprintf("%s:%d", snap.instance.ListenAddr, snap.instance.ListenPort),
			paused: snap.instance.Paused,
		}
	}
	e.mu.RUnlock()

	e.listenerMu.Lock()
	defer e.listenerMu.Unlock()

	for eid := range e.listenerErr {
		if _, ok := want[eid]; !ok {
			delete(e.listenerErr, eid)
		}
	}

	for id, cancel := range e.listeners {
		w, ok := want[id]
		if !ok || w.paused || e.listenerBind[id] != w.bind {
			cancel()
			delete(e.listeners, id)
			delete(e.listenerBind, id)
		}
	}

	time.Sleep(150 * time.Millisecond)

	for id, w := range want {
		if w.paused {
			delete(e.listenerErr, id)
			continue
		}
		if _, ok := e.listeners[id]; ok && e.listenerBind[id] == w.bind {
			continue
		}
		if can, ok := e.listeners[id]; ok {
			can()
			delete(e.listeners, id)
			delete(e.listenerBind, id)
		}

		bind := w.bind
		lctx, cancel := context.WithCancel(ctx)
		ready := make(chan error, 1)
		e.wg.Add(1)
		go func(instID int32, bindAddr string) {
			defer e.wg.Done()
			pc, err := net.ListenPacket("udp", bindAddr)
			if err != nil {
				ready <- &BindError{Bind: bindAddr, Err: err}
				return
			}
			ready <- nil
			defer pc.Close()
			e.log.Info("dns listening", "bind", bindAddr, "instance_id", instID)
			e.serveUDPLoop(lctx, pc, instID)
		}(id, bind)

		select {
		case err := <-ready:
			if err != nil {
				cancel()
				msg := err.Error()
				var be *BindError
				if errors.As(err, &be) && be.Err != nil {
					msg = be.Err.Error()
				}
				prevMsg := e.listenerErr[id]
				e.listenerErr[id] = msg
				// 定时 sync 会反复重试绑定；错误未变时不再刷 Error，避免每 ~5s 一条重复日志
				if prevMsg != msg {
					e.log.Error("udp listen", "bind", bind, "instance_id", id, "err", err)
				}
				continue
			}
		case <-ctx.Done():
			cancel()
			return ctx.Err()
		}
		delete(e.listenerErr, id)
		e.listeners[id] = cancel
		e.listenerBind[id] = bind
	}
	return nil
}

func (e *Engine) serveUDPLoop(ctx context.Context, pc net.PacketConn, instID int32) {
	buf := make([]byte, mdns.MaxMsgSize)
	sem := make(chan struct{}, e.maxConcurrentQueries)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_ = pc.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, remote, err := pc.ReadFrom(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			if ctx.Err() != nil {
				return
			}
			continue
		}
		msg := new(mdns.Msg)
		if err := msg.Unpack(buf[:n]); err != nil {
			continue
		}
		e.mu.RLock()
		snap := e.snaps[instID]
		e.mu.RUnlock()
		if snap == nil || snap.instance.Paused {
			continue
		}
		select {
		case sem <- struct{}{}:
		default:
			busy := new(mdns.Msg)
			busy.SetReply(msg)
			busy.Rcode = mdns.RcodeServerFailure
			if packed, packErr := busy.Pack(); packErr == nil {
				_, _ = pc.WriteTo(packed, remote)
			}
			continue
		}
		e.udpHandlersWG.Add(1)
		go func(snap *instanceSnap, remote net.Addr, req *mdns.Msg) {
			defer e.udpHandlersWG.Done()
			defer func() {
				<-sem
			}()
			defer func() {
				if r := recover(); r != nil {
					e.log.Error("dns udp handler panic", "instance_id", instID, "panic", r)
				}
			}()
			start := time.Now()
			out := e.handleQuery(ctx, snap, remote, req, start)
			if out == nil {
				return
			}
			packed, err := out.Pack()
			if err != nil {
				return
			}
			_, _ = pc.WriteTo(packed, remote)
		}(snap, remote, msg)
	}
}

// SyncNow reloads DB and UDP listeners (call after API mutations).
func (e *Engine) SyncNow(ctx context.Context) error {
	if err := e.Reload(ctx); err != nil {
		return err
	}
	rc := e.rootCtx
	if rc == nil {
		rc = context.Background()
	}
	return e.syncListeners(rc)
}

// ListenerErrors 返回当前 UDP 绑定失败的实例及错误信息（如端口占用）；无错误时可能返回 nil。
func (e *Engine) ListenerErrors() map[int32]string {
	e.listenerMu.Lock()
	defer e.listenerMu.Unlock()
	if len(e.listenerErr) == 0 {
		return nil
	}
	out := make(map[int32]string, len(e.listenerErr))
	for k, v := range e.listenerErr {
		out[k] = v
	}
	return out
}

// ListCacheEntries 返回当前内存 LRU 中的有效缓存条目（供管理 API）。
func (e *Engine) ListCacheEntries() []CacheListItem {
	if e == nil || e.cache == nil {
		return nil
	}
	return e.cache.listEntries()
}

func (e *Engine) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
	e.listenerMu.Lock()
	for id, c := range e.listeners {
		c()
		delete(e.listeners, id)
		delete(e.listenerBind, id)
	}
	e.listenerMu.Unlock()
	e.wg.Wait()
	e.udpHandlersWG.Wait()
	e.queryLogWG.Wait()
}
