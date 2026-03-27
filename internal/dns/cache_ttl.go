package dns

import (
	"time"

	mdns "github.com/miekg/dns"
)

// ageCachedMsgTTLs 按入缓存以来经过的整秒数，逐条扣减 Answer / Authority / Additional
// 中各 RR 的 TTL（与常见递归解析器对外语义一致；不同原始 TTL 可保持差异）。
func ageCachedMsgTTLs(msg *mdns.Msg, cachedAt, now time.Time) {
	if msg == nil {
		return
	}
	ageSec := cacheAgeSeconds(cachedAt, now)
	if ageSec == 0 {
		return
	}
	for _, rr := range msg.Answer {
		decrementRRTTLByAge(rr, ageSec)
	}
	for _, rr := range msg.Ns {
		decrementRRTTLByAge(rr, ageSec)
	}
	for _, rr := range msg.Extra {
		decrementRRTTLByAge(rr, ageSec)
	}
}

func decrementRRTTLByAge(rr mdns.RR, ageSec uint32) {
	if rr == nil {
		return
	}
	h := rr.Header()
	if ageSec >= h.Ttl {
		h.Ttl = 0
	} else {
		h.Ttl = h.Ttl - ageSec
	}
}

func cacheAgeSeconds(cachedAt, now time.Time) uint32 {
	if !now.After(cachedAt) {
		return 0
	}
	sec := int64(now.Sub(cachedAt) / time.Second)
	if sec <= 0 {
		return 0
	}
	if sec > int64(^uint32(0)) {
		return ^uint32(0)
	}
	return uint32(sec)
}
