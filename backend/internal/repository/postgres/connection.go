package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Yugsolanki/standfor-me/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func NewConnection(ctx context.Context, cfg config.DatabaseConfig) (*sqlx.DB, error) {
	// connect to postgres
	db, err := sqlx.ConnectContext(ctx, "pgx", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// set connection pool settings
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxIdleTime(cfg.MaxConnIdleTime)
	db.SetConnMaxLifetime(cfg.MaxConnLifetime)

	// ping with retry
	if err := pingWithRetry(ctx, db, 3, 2*time.Second); err != nil {
		db.Close()
		return nil, fmt.Errorf("database health check failed: %w", err)
	}

	// start pool health monitoring if interval is configured
	if cfg.HealthCheckInterval > 0 {
		go monitorPoolHealth(db, cfg.HealthCheckInterval, *slog.Default())
	}

	slog.Info("database connection pool initialized",
		slog.Group("postgres",
			slog.Int("max_open_conns", cfg.MaxOpenConns),
			slog.Int("max_idle_conns", cfg.MaxIdleConns),
			slog.Duration("max_conn_idle_time", cfg.MaxConnIdleTime),
			slog.Duration("max_conn_lifetime", cfg.MaxConnLifetime),
		),
	)

	return db, nil
}

func pingWithRetry(ctx context.Context, db *sqlx.DB, maxRetries int, delay time.Duration) error {
	var err error
	for range maxRetries {
		err = db.PingContext(ctx)
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			continue
		}
	}
	return err
}

func monitorPoolHealth(db *sqlx.DB, interval time.Duration, logger slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		stats := db.Stats()
		logger.Debug("pool stats",
			slog.Int("open_conns", stats.OpenConnections),
			slog.Int("idle_conns", stats.Idle),
			slog.Int("in_use", stats.InUse),
			slog.Int("wait_count", int(stats.WaitCount)),
			slog.Duration("wait_duration", stats.WaitDuration),
		)
	}
}
