package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/Yugsolanki/standfor-me/internal/config"
	"github.com/Yugsolanki/standfor-me/internal/middleware"
	"github.com/Yugsolanki/standfor-me/internal/middleware/ratelimit"
	internaljwt "github.com/Yugsolanki/standfor-me/internal/pkg/jwt"
	"github.com/Yugsolanki/standfor-me/internal/pkg/response"
	appvalidator "github.com/Yugsolanki/standfor-me/internal/pkg/validator"
	"github.com/Yugsolanki/standfor-me/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

type Services struct {
	Auth *service.AuthService
	User *service.UserService
}

type Server struct {
	cfg             *config.ServerConfig
	logger          *slog.Logger
	router          *chi.Mux
	services        *Services
	http            *http.Server
	redis           *redis.Client
	rateLimitConfig *config.RateLimitConfig
	validator       *appvalidator.Validator
	jwtSvc          *internaljwt.Service
}

func New(
	cfg *config.ServerConfig,
	logger *slog.Logger,
	svcs *Services,
	redisClient *redis.Client,
	rateLimitConfig *config.RateLimitConfig,
	validator *appvalidator.Validator,
	jwtSvc *internaljwt.Service,
) *Server {
	s := &Server{
		cfg:             cfg,
		logger:          logger,
		router:          chi.NewRouter(),
		services:        svcs,
		redis:           redisClient,
		rateLimitConfig: rateLimitConfig,
		validator:       validator,
		jwtSvc:          jwtSvc,
	}

	// Wire middleware
	s.setupMiddleware(cfg)

	// Setup conditional rate limiters
	cl, err := s.setupRateLimiters(s.redis, s.logger)
	if err != nil {
		return nil
	}
	s.router.Use(cl.Handler)

	// Setup routes
	s.setupRoutes(jwtSvc)

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

func (s *Server) setupMiddleware(cfg *config.ServerConfig) {
	env := os.Getenv("APP_ENV")

	// Request ID middleware
	s.router.Use(middleware.RequestID)

	// Recoverer
	s.router.Use(middleware.Recoverer(s.logger))

	// Global Rate Limiter
	s.router.Use(s.globalRateLimiter())
	// Payload Size Limiter
	s.router.Use(middleware.PayloadLimit(middleware.DefaultMaxBodySize))

	// Compress
	s.router.Use(middleware.Compress)

	// Canonical Logger
	s.router.Use(middleware.CanonicalLogger(s.logger))

	// Security Headers
	if env == "development" {
		s.router.Use(middleware.CORS(middleware.DevelopmentCORSConfig()))
		s.router.Use(middleware.SecurityHeaders(middleware.DevelopmentSecurityHeadersConfig()))
	} else {
		s.router.Use(middleware.CORS(middleware.DefaultCORSConfig()))
		s.router.Use(middleware.SecurityHeaders(middleware.DefaultSecurityHeadersConfig()))
	}
	// CSP Nonce
	s.router.Use(middleware.CSPNonce())

	// Timeout
	s.router.Use(middleware.Timeout(cfg.RequestTimeout))

	// Log
	s.logger.Info("middleware has been established",
		"app_env", os.Getenv("APP_ENV"),
		"request_timeout", cfg.RequestTimeout,
		"max_body_size", middleware.DefaultMaxBodySize,
		"global_rate_limit", s.rateLimitConfig.Global.Limit,
		"global_rate_window", s.rateLimitConfig.Global.Window,
	)
}

// rootHandler handles the root (/) request.
func (s *Server) rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response.JSONMessage(w, r, http.StatusOK, "OK")
}

// healthHandler handles the health (/health) check request.
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response.JSONMessage(w, r, http.StatusOK, "OK")
}

func (s *Server) globalRateLimiter() func(http.Handler) http.Handler {
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
		// Return a no-op middleware that just passes through if the limiter fails to initialize
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	s.logger.Info("ratelimiter has been established")
	return globalLimiter.Handler
}
