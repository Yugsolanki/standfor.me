package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Yugsolanki/standfor-me/internal/config"
	"github.com/go-chi/chi/v5"
)

type Services struct {
	Auth any
	User any
}

type Server struct {
	cfg      *config.ServerConfig
	logger   *slog.Logger
	router   *chi.Mux
	services *Services
	http     *http.Server
}

func New(cfg *config.ServerConfig, logger *slog.Logger, services *Services) *Server {
	s := &Server{
		cfg:      cfg,
		logger:   logger,
		router:   chi.NewRouter(),
		services: services,
	}

	// Build handler structs
	// TODO

	// Wire middleware + routes
	// TODO

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
