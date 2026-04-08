package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Yugsolanki/standfor-me/internal/config"
	internaljwt "github.com/Yugsolanki/standfor-me/internal/pkg/jwt"
	appvalidator "github.com/Yugsolanki/standfor-me/internal/pkg/validator"
	"github.com/Yugsolanki/standfor-me/internal/repository/postgres"
	"github.com/Yugsolanki/standfor-me/internal/repository/redis"
	"github.com/Yugsolanki/standfor-me/internal/server"
	"github.com/Yugsolanki/standfor-me/internal/service"
)

func main() {
	// --- Structured Logging ---
	logger := initLogger()
	slog.SetDefault(logger)

	// --- Load Configuration ---
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configurations", "error", err)
		os.Exit(1)
	}
	slog.Info("configuration loaded", "environment", cfg.Env, "port", cfg.Server.Port)

	// --- Database Connection ---
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := postgres.NewConnection(ctx, cfg.Database)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	slog.Info("database connection established")

	// --- Redis Connection ---
	redisClient, err := redis.NewRedisClient(cfg.Redis)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	slog.Info("redis connection established")

	// --- Repositories ---
	userRepo := postgres.NewUserRepository(db)
	refreshTokenRepo := postgres.NewRefreshTokenRepository(db)

	// --- Application Services ---
	jwtSvc := internaljwt.New(cfg.JWT)

	authSvc := service.NewAuthService(userRepo, refreshTokenRepo, jwtSvc)
	userSvc := service.NewUserService(userRepo, refreshTokenRepo)

	// --- Validator
	validator := appvalidator.New()

	// --- Server
	srv := server.New(
		&cfg.Server,
		logger,
		&server.Services{
			Auth: authSvc,
			User: userSvc,
		},
		redisClient,
		&cfg.RateLimit,
		validator,
		jwtSvc,
	)

	// --- Graceful Shutdown
	go func() {
		slog.Info("server starting", "port", cfg.Server.Port)
		if err := srv.Start(); err != nil {
			slog.Error("server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// --- Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	slog.Info("shutdown signal received", "signal", sig.String())

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server gracefully stopped")
}

func initLogger() *slog.Logger {
	env := os.Getenv("APP_ENV")

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}

	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
