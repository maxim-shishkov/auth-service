package http

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/maxim-shishkov/auth-service/internal/domain"
)

type TokenParser interface {
	Parse(tokenString string) (uuid.UUID, error)
}

type ctxKey int

const userIDCtxKey ctxKey = iota

func userIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDCtxKey).(uuid.UUID)
	return id, ok
}

func authMiddleware(parser TokenParser) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")

			const prefix = "Bearer "
			if !strings.HasPrefix(header, prefix) {
				writeError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}

			rawToken := strings.TrimSpace(strings.TrimPrefix(header, prefix))
			if rawToken == "" {
				writeError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}

			userID, err := parser.Parse(rawToken)
			if err != nil {
				status := http.StatusUnauthorized
				msg := "invalid token"

				if errors.Is(err, domain.ErrTokenExpired) {
					msg = "token expired"
				}

				writeError(w, status, msg)
				return
			}

			ctx := context.WithValue(r.Context(), userIDCtxKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func recoverMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic recovered", "panic", rec, "path", r.URL.Path)
					writeError(w, http.StatusInternalServerError, "internal server error")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

func rateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				key = r.RemoteAddr
			}
			if !limiter.Allow(key) {
				w.Header().Set("Retry-After", "60")
				writeError(
					w,
					http.StatusTooManyRequests,
					"too many requests",
				)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
