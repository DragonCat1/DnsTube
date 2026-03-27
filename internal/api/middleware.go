package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/dnstube/dnstube/internal/auth"
)

type ctxKey int

const userIDKey ctxKey = 1
const usernameKey ctxKey = 2

func UserIDFromContext(ctx context.Context) (int32, bool) {
	v := ctx.Value(userIDKey)
	if v == nil {
		return 0, false
	}
	id, ok := v.(int32)
	return id, ok
}

// UsernameFromContext JWT 中的用户名（requireAuth 后可用）。
func UsernameFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(usernameKey)
	if v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		o := r.Header.Get("Origin")
		if o != "" {
			w.Header().Set("Access-Control-Allow-Origin", o)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requireAuth(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			writeObjErr(w, http.StatusUnauthorized, CodeMissingBearer)
			return
		}
		tok := strings.TrimSpace(strings.TrimPrefix(h, "Bearer"))
		c, err := auth.ParseJWT(secret, tok)
		if err != nil {
			writeObjErr(w, http.StatusUnauthorized, CodeInvalidToken)
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, c.UserID)
		ctx = context.WithValue(ctx, usernameKey, c.Username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
