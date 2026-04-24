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
	meilirepository "github.com/Yugsolanki/standfor-me/internal/repository/meilisearch"
	"github.com/Yugsolanki/standfor-me/internal/repository/postgres"
	"github.com/Yugsolanki/standfor-me/internal/repository/redis"
	"github.com/Yugsolanki/standfor-me/internal/server"
	"github.com/Yugsolanki/standfor-me/internal/service"
	searchservice "github.com/Yugsolanki/standfor-me/internal/service/search"
)

// @title			Standfor API
// @version		1.0
// @description	Standfor API
// @host			localhost:8080
// @BasePath		/api/v1
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

	// --- Meilisearch Config
	meiliCfg, err := config.LoadSearchConfig()
	if err != nil {
		slog.Error("failed to load meilisearch config", "error", err)
		os.Exit(1)
	}

	// --- Meilisearch Client
	meiliClient, err := meilirepository.NewClient(&meiliCfg.Meilisearch, logger)
	if err != nil {
		slog.Error("failed to create meilisearch client", "error", err)
		os.Exit(1)
	}

	// --- Initialize Meilisearch Indexes
	if err := meiliClient.Initialize(context.Background()); err != nil {
		slog.Error("failed to initialize meilisearch indexes", "error", err)
		os.Exit(1)
	}

	// --- Data Repository (for indexing) ---
	movementDataRepo := postgres.NewMovementIndexingRepository(db)
	userDataRepo := postgres.NewUserIndexingRepository(db)
	orgDataRepo := postgres.NewOrganizationIndexingRepository(db)

	// --- Search Service ---
	searchSvc := searchservice.NewService(
		meilirepository.NewMovementRepository(meiliClient),
		meilirepository.NewUserRepository(meiliClient),
		meilirepository.NewOrganizationRepository(meiliClient),
		movementDataRepo,
		userDataRepo,
		orgDataRepo,
		logger,
	)

	// --- Validator
	validator := appvalidator.New()

	// --- Server
	srv := server.New(
		&cfg.Server,
		logger,
		&server.Services{
			Auth:   authSvc,
			User:   userSvc,
			Search: searchSvc,
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
