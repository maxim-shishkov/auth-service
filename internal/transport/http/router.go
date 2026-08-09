package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func NewRouter(h *Handler, parser TokenParser, log *slog.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(recoverMiddleware(log))
	r.Use(chimw.Timeout(10 * time.Second))

	loginLimiter := NewRateLimiter(5, time.Minute)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Post("/register", h.Register)

	r.With(rateLimitMiddleware(loginLimiter)).
		Post("/login", h.Login)

	r.Group(func(protected chi.Router) {
		protected.Use(authMiddleware(parser))
		protected.Get("/me", h.Me)
	})

	return r
}
