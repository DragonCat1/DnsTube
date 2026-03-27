package api

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/dnstube/dnstube/internal/store"
)

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// insertAuditLog 异步写入审计日志（不阻塞请求）。
func (s *Server) insertAuditLog(r *http.Request, level, event, username, message string, meta map[string]any) {
	if level == "" {
		level = "info"
	}
	ip := clientIP(r)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		var ev *string
		if event != "" {
			ev = &event
		}
		var u *string
		if username != "" {
			u = &username
		}
		ipStr := ip
		var ipPtr *string
		if ipStr != "" {
			ipPtr = &ipStr
		}
		_ = s.Store.InsertLogEntry(ctx, store.LogKindAudit, level, ev, message, u, ipPtr, meta)
		if s.Notify != nil {
			s.Notify.MaybeAudit(event, level, message, u)
		}
	}()
}
