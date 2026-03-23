package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/FXAZfung/random-image/internal/cache"
	"github.com/FXAZfung/random-image/internal/config"
	"github.com/FXAZfung/random-image/internal/limiter"
	"github.com/FXAZfung/random-image/internal/picker"
)

type Server struct {
	httpServer *http.Server
}

func New(cfg *config.Config, p *picker.Picker, c *cache.Cache, lim *limiter.Limiter, version string) *Server {
	handler := NewHandler(p, c, lim, cfg.Relay, version)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.HandleIndex)
	mux.HandleFunc("/health", handler.HandleHealth)
	mux.HandleFunc("/api/categories", handler.HandleCategories)
	mux.HandleFunc("/api/", handler.HandleRandom)

	var h http.Handler = mux
	h = corsMiddleware(h)
	if cfg.Limiter.Enabled {
		h = rateLimitMiddleware(lim)(h)
	}
	h = loggingMiddleware(h)
	h = recoveryMiddleware(h)

	httpServer := &http.Server{
		Addr:         cfg.Server.Address,
		Handler:      h,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	return &Server{httpServer: httpServer}
}

func (s *Server) Start() error {
	slog.Info("server starting", "address", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("server shutting down")
	return s.httpServer.Shutdown(ctx)
}
