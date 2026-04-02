package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/Yugsolanki/standfor-me/internal/config"
	"github.com/Yugsolanki/standfor-me/internal/repository/postgres"
	"github.com/hibiken/asynq"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("starting standfor.me background worker")

	// Load Configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Database Connection with timeout
	connCtx, connCancel := context.WithTimeout(context.Background(), 10*time.Second)
	db, err := postgres.NewConnection(connCtx, cfg.Database)
	connCancel() // Cancel the context after connection attempt
	if err != nil {
		slog.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("connected to database")

	// Asynq Server Setup
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
	}

	// Create Asynq server
	srv := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: cfg.Worker.Concurrency,
			RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
				return asynq.DefaultRetryDelayFunc(n, e, t)
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				logger.Error("failed to process task", slog.String("task", task.Type()), slog.String("error", err.Error()))
			}),
			LogLevel: asynq.InfoLevel,
		},
	)

	// Create a new ServeMux
	mux := asynq.NewServeMux()
	// Register handlers
	// mux.HandleFunc(task.TypeWelcomeEmail, tasks.HandleWelcomeEmailTask)

	logger.Info("starting asynq server with concurrency", slog.Int("concurrency", cfg.Worker.Concurrency))

	if err := srv.Run(mux); err != nil {
		logger.Error("failed to run asynq server", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("worker shut down gracefully")
}
