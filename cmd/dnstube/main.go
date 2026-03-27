package main

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/dnstube/dnstube/internal/api"
	"github.com/dnstube/dnstube/internal/applog"
	"github.com/dnstube/dnstube/internal/auth"
	"github.com/dnstube/dnstube/internal/config"
	dnsengine "github.com/dnstube/dnstube/internal/dns"
	"github.com/dnstube/dnstube/internal/notify"
	"github.com/dnstube/dnstube/internal/store"
)

//go:embed all:web/dist
var webDist embed.FS

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	if err := store.MigrateUp(cfg.DatabaseURL); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}

	n := notify.New(st)
	applog.SetDefault(st, slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}), n)

	if err := bootstrapAdmin(ctx, st); err != nil {
		slog.Error("bootstrap admin", "err", err)
		os.Exit(1)
	}

	eng := dnsengine.NewEngine(
		st,
		slog.Default(),
		cfg.DNSUDPUpstreamTimeout,
		cfg.DNSUDPQueryTotalTimeout,
		cfg.DNSUDPMaxConcurrentQuery,
	)
	if err := eng.Start(ctx); err != nil {
		slog.Error("dns engine", "err", err)
		os.Exit(1)
	}

	go func() {
		tick := time.NewTicker(24 * time.Hour)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				if err := eng.RefreshAllURLPatterns(context.Background()); err != nil {
					slog.Error("scheduled url pattern refresh", "err", err)
				}
			}
		}
	}()

	sub, err := fs.Sub(webDist, "web/dist")
	if err != nil {
		slog.Error("embed static", "err", err)
		os.Exit(1)
	}
	static := spaFileServer(sub)

	srv := &api.Server{
		Store:  st,
		Engine: eng,
		Secret: cfg.JWTSecret,
		Log:    slog.Default(),
		Notify: n,
	}
	h := srv.Handler(static)

	httpSrv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		slog.Info("http listening", "addr", cfg.HTTPAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	_ = httpSrv.Shutdown(shCtx)
	eng.Stop()
}

// spaFileServer serves embedded web/dist and falls back to index.html for client-side routes (Vue history mode).
func spaFileServer(root fs.FS) http.Handler {
	fsSrv := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			fsSrv.ServeHTTP(w, r)
			return
		}
		p := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if p != "" {
			if _, err := fs.Stat(root, p); err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					r2 := r.Clone(r.Context())
					r2.URL.Path = "/"
					fsSrv.ServeHTTP(w, r2)
					return
				}
			}
		}
		fsSrv.ServeHTTP(w, r)
	})
}

func bootstrapAdmin(ctx context.Context, st *store.Store) error {
	n, err := st.CountAdmins(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	const pw = "admin"
	hash, err := auth.HashPassword(pw)
	if err != nil {
		return err
	}
	if _, err := st.CreateAdmin(ctx, "admin", hash); err != nil {
		return err
	}
	slog.Warn("=== INITIAL ADMIN USER: admin ===")
	slog.Warn("=== INITIAL ADMIN PASSWORD (change after login) ===", "password", pw)
	return nil
}
