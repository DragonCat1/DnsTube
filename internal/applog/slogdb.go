package applog

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/dnstube/dnstube/internal/notify"
	"github.com/dnstube/dnstube/internal/store"
)

// Handler 将 slog 输出到 stderr（inner），并把 Info 及以上级别异步写入 system_logs（kind=system）。
type Handler struct {
	inner    slog.Handler
	store    *store.Store
	minLevel slog.Level
	notify   *notify.Notifier
}

// NewHandler 创建 DB 双写 Handler；minLevel 通常为 slog.LevelInfo；notify 可选，用于 Telegram。
func NewHandler(inner slog.Handler, st *store.Store, minLevel slog.Level, n *notify.Notifier) *Handler {
	if minLevel == 0 {
		minLevel = slog.LevelInfo
	}
	return &Handler{inner: inner, store: st, minLevel: minLevel, notify: n}
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	err := h.inner.Handle(ctx, r)
	if r.Level < h.minLevel || h.store == nil {
		return err
	}
	msg := formatRecordMessage(r)
	meta := attrsToMeta(r)
	if strings.TrimSpace(msg) == "" {
		return err
	}
	lvl := slogLevelString(r.Level)
	go func() {
		ctx2, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = h.store.InsertLogEntry(ctx2, store.LogKindSystem, lvl, nil, msg, nil, nil, meta)
		if h.notify != nil {
			h.notify.MaybeSystem(lvl, msg)
		}
	}()
	return err
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{inner: h.inner.WithAttrs(attrs), store: h.store, minLevel: h.minLevel, notify: h.notify}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{inner: h.inner.WithGroup(name), store: h.store, minLevel: h.minLevel, notify: h.notify}
}

func formatRecordMessage(r slog.Record) string {
	var b strings.Builder
	b.WriteString(r.Message)
	r.Attrs(func(a slog.Attr) bool {
		b.WriteByte(' ')
		lk := strings.ToLower(a.Key)
		if strings.Contains(lk, "password") || strings.Contains(lk, "token") || strings.Contains(lk, "secret") {
			b.WriteString(a.Key)
			b.WriteString("=[redacted]")
		} else {
			b.WriteString(a.String())
		}
		return true
	})
	return b.String()
}

func attrsToMeta(r slog.Record) map[string]any {
	m := make(map[string]any)
	n := 0
	r.Attrs(func(a slog.Attr) bool {
		k := a.Key
		v := a.Value.Any()
		lk := strings.ToLower(k)
		if strings.Contains(lk, "password") || strings.Contains(lk, "token") || strings.Contains(lk, "secret") {
			m[k] = "[redacted]"
		} else {
			m[k] = v
		}
		n++
		return true
	})
	if n == 0 {
		return nil
	}
	return m
}

func slogLevelString(l slog.Level) string {
	switch {
	case l < slog.LevelInfo:
		return "debug"
	case l < slog.LevelWarn:
		return "info"
	case l < slog.LevelError:
		return "warn"
	default:
		return "error"
	}
}

var _ slog.Handler = (*Handler)(nil)

// SetDefault 将 slog 默认 logger 设为 stderr + DB 双写（store 非 nil 时）；notify 可选。
func SetDefault(st *store.Store, inner slog.Handler, n *notify.Notifier) {
	if st == nil {
		slog.SetDefault(slog.New(inner))
		return
	}
	h := NewHandler(inner, st, slog.LevelInfo, n)
	slog.SetDefault(slog.New(h))
}
