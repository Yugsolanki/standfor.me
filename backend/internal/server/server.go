package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Yugsolanki/standfor-me/internal/config"
	"github.com/Yugsolanki/standfor-me/internal/middleware/ratelimit"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

type Services struct {
	Auth any
	User any
}

type Server struct {
	cfg             *config.ServerConfig
	logger          *slog.Logger
	router          *chi.Mux
	services        *Services
	http            *http.Server
	redis           *redis.Client
	rateLimitConfig *config.RateLimitConfig
}

func New(cfg *config.ServerConfig, logger *slog.Logger, services *Services, redis *redis.Client, rateLimitConfig *config.RateLimitConfig) *Server {
	s := &Server{
		cfg:             cfg,
		logger:          logger,
		router:          chi.NewRouter(),
		services:        services,
		redis:           redis,
		rateLimitConfig: rateLimitConfig,
	}

	// Build handler structs

	// Wire middleware
	s.setupMiddleware()

	// Wire routes
	s.router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	s.http = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      s.router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	return s
}

func (s *Server) Start() error {
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

func (s *Server) Router() *chi.Mux {
	return s.router
}

func (s *Server) setupMiddleware() {
	// Global Rate Limiter
	globalLimiter, err := ratelimit.New(s.redis, ratelimit.Config{
		Limit:      s.rateLimitConfig.Global.Limit,
		Window:     s.rateLimitConfig.Global.Window,
		Strategy:   ratelimit.SlidingWindow,
		Prefix:     "standfor:rl:global",
		TrustProxy: true,
		SkipFunc: func(r *http.Request) bool {
			return r.URL.Path == "/health" || r.URL.Path == "/metrics"
		},
		FallbackToAllow: true,
	}, s.logger)

	if err != nil {
		s.logger.Error("failed to create global rate limiter", "error", err)
	}
	if err == nil {
		s.logger.Info("ratelimiter has been established")
	}
	s.router.Use(globalLimiter.Handler)
}
