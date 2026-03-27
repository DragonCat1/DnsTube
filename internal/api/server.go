package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/dnstube/dnstube/internal/dns"
	"github.com/dnstube/dnstube/internal/notify"
	"github.com/dnstube/dnstube/internal/store"
	"github.com/go-chi/chi/v5"
)

type EngineSync interface {
	SyncNow(ctx context.Context) error
	ListenerErrors() map[int32]string
	ListCacheEntries() []dns.CacheListItem
}

type Server struct {
	Store  *store.Store
	Engine EngineSync
	Secret string
	Log    *slog.Logger
	Notify *notify.Notifier
}

func (s *Server) Handler(static http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(cors)
	r.Get("/healthz", s.healthz)
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", s.login)
		r.Group(func(r chi.Router) {
			r.Use(func(next http.Handler) http.Handler {
				return requireAuth(s.Secret, next)
			})
			r.Get("/auth/me", s.me)
			r.Patch("/auth/profile", s.patchProfile)

			r.Get("/settings/system", s.getSystemSettings)
			r.Patch("/settings/system", s.patchSystemSettings)
			r.Post("/settings/system/telegram-test", s.postTelegramTest)

			r.Get("/instances", s.listInstances)
			r.Post("/instances", s.createInstance)
			r.Get("/instances/{id}", s.getInstance)
			r.Patch("/instances/{id}", s.patchInstance)
			r.Delete("/instances/{id}", s.deleteInstance)

			r.Get("/instances/{id}/records", s.listRecords)
			r.Post("/instances/{id}/records", s.createRecord)
			r.Patch("/records/{id}", s.patchRecord)
			r.Delete("/records/{id}", s.deleteRecord)

			r.Get("/upstream-groups", s.listGroups)
			r.Get("/upstream-groups/{id}/servers", s.listGroupServers)
			r.Post("/upstream-groups", s.createGroup)
			r.Patch("/upstream-groups/{id}", s.patchGroup)
			r.Delete("/upstream-groups/{id}", s.deleteGroup)
			r.Post("/upstream-groups/{id}/servers", s.addServer)
			r.Delete("/upstream-servers/{id}", s.deleteServer)

			r.Get("/instances/{id}/forward-rules", s.listRules)
			r.Post("/instances/{id}/forward-rules", s.createRule)
			r.Post("/forward-rules/{id}/refresh-pattern", s.refreshRulePattern)
			r.Get("/forward-rules/{id}/pattern-entries", s.getForwardRulePatternEntries)
			r.Patch("/forward-rules/{id}/disabled", s.patchRuleDisabled)
			r.Patch("/forward-rules/{id}", s.patchRule)
			r.Delete("/forward-rules/{id}", s.deleteRule)

			r.Get("/query-logs", s.listQueryLogs)
			r.Post("/query-logs/delete", s.deleteQueryLogs)

			r.Get("/system-logs", s.listSystemLogs)
			r.Post("/system-logs/delete", s.deleteSystemLogs)

			r.Get("/cache-entries", s.listCacheEntries)

			r.Get("/dashboard/stats", s.dashboardStats)
			r.Get("/dashboard/summary", s.dashboardSummary)
		})
	})
	if static != nil {
		r.Handle("/*", static)
	}
	return r
}
