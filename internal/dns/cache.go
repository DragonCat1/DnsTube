package dns

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	mdns "github.com/miekg/dns"
)

type cacheEntry struct {
	msg       *mdns.Msg
	expiresAt time.Time
	cachedAt  time.Time
	ttl       time.Duration // 写入时采用的缓存 TTL（与应答 minTTL 一致）
}

type queryCache struct {
	mu  sync.Mutex
	lru *lru.Cache[string, *cacheEntry]
	ttl time.Duration
}

func newQueryCache(size int, defaultTTL time.Duration) *queryCache {
	if size <= 0 {
		size = 10000
	}
	if defaultTTL <= 0 {
		defaultTTL = 5 * time.Minute
	}
	c, _ := lru.New[string, *cacheEntry](size)
	return &queryCache{lru: c, ttl: defaultTTL}
}

func cacheKey(instanceID int32, qname string, qtype uint16) string {
	return strconv.FormatInt(int64(instanceID), 10) + "|" + qname + "|" + strconv.FormatUint(uint64(qtype), 10)
}

func (c *queryCache) get(key string) (*mdns.Msg, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.lru.Get(key)
	if !ok || e == nil {
		return nil, false
	}
	now := time.Now()
	if now.After(e.expiresAt) {
		c.lru.Remove(key)
		return nil, false
	}
	msg := e.msg.Copy()
	ageCachedMsgTTLs(msg, e.cachedAt, now)
	return msg, true
}

func (c *queryCache) set(key string, msg *mdns.Msg, ttl time.Duration) {
	if ttl <= 0 {
		ttl = c.ttl
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	c.lru.Add(key, &cacheEntry{msg: msg.Copy(), expiresAt: now.Add(ttl), cachedAt: now, ttl: ttl})
}

// CacheListItem 管理 API 列出的单条缓存（内存 LRU）。
type CacheListItem struct {
	InstanceID    int32      `json:"instance_id"`
	Qname         string     `json:"qname"`
	Qtype         string     `json:"qtype"`
	TTLSeconds    int        `json:"ttl_seconds"`
	CachedAt      *time.Time `json:"cached_at,omitempty"`
	ExpiresAt     time.Time  `json:"expires_at"`
	ResultSummary string     `json:"result_summary"`
}

func (c *queryCache) listEntries() []CacheListItem {
	c.mu.Lock()
	defer c.mu.Unlock()
	keys := c.lru.Keys()
	now := time.Now()
	var out []CacheListItem
	for _, k := range keys {
		e, ok := c.lru.Get(k)
		if !ok || e == nil {
			continue
		}
		if now.After(e.expiresAt) {
			continue
		}
		parts := strings.SplitN(k, "|", 3)
		if len(parts) != 3 {
			continue
		}
		iid, err := strconv.ParseInt(parts[0], 10, 32)
		if err != nil {
			continue
		}
		qt, err := strconv.ParseUint(parts[2], 10, 16)
		if err != nil {
			continue
		}
		rem := int(time.Until(e.expiresAt) / time.Second)
		if rem < 0 {
			rem = 0
		}
		item := CacheListItem{
			InstanceID:    int32(iid),
			Qname:         parts[1],
			Qtype:         qtypeString(uint16(qt)),
			TTLSeconds:    rem,
			ExpiresAt:     e.expiresAt,
			ResultSummary: AnswerSummaryForLog(e.msg),
		}
		if !e.cachedAt.IsZero() {
			ca := e.cachedAt
			item.CachedAt = &ca
		}
		out = append(out, item)
	}
	return out
}

// cmpCacheListTiebreak 缓存项无 DB id，用实例 + 域名 + 类型作为全序次要键。
func cmpCacheListTiebreak(a, b CacheListItem) int {
	if a.InstanceID != b.InstanceID {
		return int(a.InstanceID - b.InstanceID)
	}
	c := strings.Compare(a.Qname, b.Qname)
	if c != 0 {
		return c
	}
	return strings.Compare(a.Qtype, b.Qtype)
}

// SortCacheList 按字段排序（用于 API）。
func SortCacheList(items []CacheListItem, sortBy, order string) {
	less := func(i, j int) bool {
		a, b := items[i], items[j]
		var cmp int
		switch sortBy {
		case "instance_id":
			cmp = int(a.InstanceID - b.InstanceID)
			if cmp == 0 {
				cmp = strings.Compare(a.Qname, b.Qname)
			}
			if cmp == 0 {
				cmp = strings.Compare(a.Qtype, b.Qtype)
			}
		case "qname":
			cmp = strings.Compare(a.Qname, b.Qname)
		case "qtype":
			cmp = strings.Compare(a.Qtype, b.Qtype)
		case "ttl_seconds":
			cmp = a.TTLSeconds - b.TTLSeconds
		case "cached_at":
			var ta, tb time.Time
			if a.CachedAt != nil {
				ta = *a.CachedAt
			}
			if b.CachedAt != nil {
				tb = *b.CachedAt
			}
			if ta.Before(tb) {
				cmp = -1
			} else if ta.After(tb) {
				cmp = 1
			}
		case "expires_at":
			if a.ExpiresAt.Before(b.ExpiresAt) {
				cmp = -1
			} else if a.ExpiresAt.After(b.ExpiresAt) {
				cmp = 1
			}
		case "result_summary":
			cmp = strings.Compare(a.ResultSummary, b.ResultSummary)
		default:
			if a.ExpiresAt.Before(b.ExpiresAt) {
				cmp = -1
			} else if a.ExpiresAt.After(b.ExpiresAt) {
				cmp = 1
			}
		}
		if cmp == 0 {
			cmp = cmpCacheListTiebreak(a, b)
		}
		if strings.EqualFold(order, "desc") {
			return cmp > 0
		}
		return cmp < 0
	}
	sort.SliceStable(items, less)
}
