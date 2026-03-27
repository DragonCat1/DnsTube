package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds process-level settings from environment.
type Config struct {
	DatabaseURL              string
	JWTSecret                string
	HTTPAddr                 string
	MigrationsPath           string // empty = use embedded migrations
	ShutdownTimeout          time.Duration
	DNSUDPUpstreamTimeout    time.Duration
	DNSUDPQueryTotalTimeout  time.Duration
	DNSUDPMaxConcurrentQuery int
}

func Load() (Config, error) {
	c := Config{
		HTTPAddr:                 getEnv("HTTP_ADDR", ":8080"),
		ShutdownTimeout:          15 * time.Second,
		DNSUDPUpstreamTimeout:    2 * time.Second,
		DNSUDPMaxConcurrentQuery: 256,
	}
	c.DatabaseURL = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	c.JWTSecret = strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if c.JWTSecret == "" {
		return c, fmt.Errorf("JWT_SECRET is required")
	}
	if v := os.Getenv("SHUTDOWN_TIMEOUT_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.ShutdownTimeout = time.Duration(n) * time.Second
		}
	}
	if v := strings.TrimSpace(os.Getenv("DNS_UDP_UPSTREAM_TIMEOUT_MS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.DNSUDPUpstreamTimeout = time.Duration(n) * time.Millisecond
		}
	}
	if v := strings.TrimSpace(os.Getenv("DNS_UDP_MAX_CONCURRENT_QUERIES")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.DNSUDPMaxConcurrentQuery = n
		}
	}
	c.DNSUDPQueryTotalTimeout = c.DNSUDPUpstreamTimeout + time.Second
	c.MigrationsPath = strings.TrimSpace(os.Getenv("MIGRATIONS_PATH"))
	return c, nil
}

func getEnv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
