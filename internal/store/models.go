package store

import (
	"encoding/json"
	"time"
)

type AdminUser struct {
	ID           int32     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type DNSInstance struct {
	ID                     int32     `json:"id"`
	Name                   string    `json:"name"`
	ListenAddr             string    `json:"listen_addr"`
	ListenPort             int       `json:"listen_port"`
	Paused                 bool      `json:"paused"`
	DefaultUpstreamGroupID *int32    `json:"default_upstream_group_id"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type DNSRecord struct {
	ID         int32     `json:"id"`
	InstanceID int32     `json:"instance_id"`
	Name       string    `json:"name"`
	Rtype      string    `json:"rtype"`
	TTL        int       `json:"ttl"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type UpstreamGroup struct {
	ID        int32     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type UpstreamServer struct {
	ID        int32  `json:"id"`
	GroupID   int32  `json:"group_id"`
	Address   string `json:"address"`
	Port      int    `json:"port"`
	SortOrder int    `json:"sort_order"`
	// Protocol 上游传输协议：udp / dot / doh。空值由 store 层归一为 "udp"。
	Protocol string `json:"protocol"`
	// Path DoH 端点路径，如 "/dns-query"；非 DoH 时为 nil。
	Path *string `json:"path,omitempty"`
	// TLSServerName DoT/DoH TLS 握手与证书校验的 SNI；为空表示沿用 Address。
	TLSServerName *string   `json:"tls_server_name,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type ForwardRule struct {
	ID            int32     `json:"id"`
	InstanceID    int32     `json:"instance_id"`
	NamePattern   string    `json:"name_pattern"`
	PatternSource string    `json:"pattern_source"`
	PatternURL    *string   `json:"pattern_url,omitempty"`
	PatternFormat string    `json:"pattern_format"`
	TargetGroupID int32     `json:"target_group_id"`
	Mode          string    `json:"mode"`
	Priority      int       `json:"priority"`
	HitCount      int64     `json:"hit_count"`
	Disabled      bool      `json:"disabled"`
	CreatedAt     time.Time `json:"created_at"`
	// PatternFetchedBody 仅引擎加载与拉取更新时使用；列表 API 不返回（json:"-"）。
	PatternFetchedBody *string `json:"-"`
	// PatternResolvedEntries autoproxy / autoproxy_base64 下各条匹配用正则模式串（re.String()）的 JSON 数组；仅 GetForwardRuleByID 等按需加载。
	PatternResolvedEntries []byte     `json:"-"`
	PatternRuleCount       int        `json:"pattern_rule_count"`
	PatternFetchedAt       *time.Time `json:"pattern_fetched_at,omitempty"`
}

type QueryLog struct {
	ID            int64     `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	InstanceID    int32     `json:"instance_id"`
	ClientIP      string    `json:"client_ip"`
	Qname         string    `json:"qname"`
	Qtype         string    `json:"qtype"`
	ResponseCode  string    `json:"response_code"`
	CacheHit      bool      `json:"cache_hit"`
	Forwarded     bool      `json:"forwarded"`
	UpstreamAddr  *string   `json:"upstream_addr"`
	UpstreamMs    *int      `json:"upstream_ms"`
	TotalMs       *int      `json:"total_ms"`
	ErrorMessage  *string   `json:"error_message"`
	ResultSummary *string   `json:"result_summary"`
	// ForwardUpstreamGroupID 实际转发使用的转发组（命中转发规则或默认转发组时；缓存命中等为 null）。
	ForwardUpstreamGroupID *int32 `json:"forward_upstream_group_id,omitempty"`
}

// QueryLogRow 列表查询 JOIN 后的展示字段。
type QueryLogRow struct {
	QueryLog
	InstanceName string `json:"instance_name"`
	// ForwardGroupName 转发组名称（无转发组时为空）。
	ForwardGroupName string `json:"forward_group_name"`
}

// SystemLog 系统/审计日志（表 system_logs）。
type SystemLog struct {
	ID        int64           `json:"id"`
	CreatedAt time.Time       `json:"created_at"`
	Kind      string          `json:"kind"`
	Level     string          `json:"level"`
	Event     *string         `json:"event,omitempty"`
	Message   string          `json:"message"`
	Username  *string         `json:"username,omitempty"`
	ClientIP  *string         `json:"client_ip,omitempty"`
	Meta      json.RawMessage `json:"meta,omitempty"`
}
