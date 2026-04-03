package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/Yugsolanki/standfor-me/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	// --- Structured Logging ---
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	}))
	slog.SetDefault(logger)

	direction := flag.String("direction", "up", "migration direction: up, down or force")
	steps := flag.Int("steps", 0, "number of migrations to run (0 = all)")
	path := flag.String("path", "./migrations", "path to migration files")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configurations", "error", err)
		os.Exit(1)
	}

	m, err := migrate.New("file://"+*path, cfg.Database.DSN())
	if err != nil {
		slog.Error("failed to create migrate instance", "error", err)
		os.Exit(1)
	}
	defer m.Close()

	switch *direction {
	case "up":
		if *steps > 0 {
			err = m.Steps(*steps)
		} else {
			err = m.Up()
		}
	case "down":
		if *steps > 0 {
			err = m.Steps(-*steps)
		} else {
			err = m.Down()
		}
	case "force":
		err = m.Force(int(*steps))
	default:
		slog.Error("invalid direction", "direction", *direction)
		os.Exit(1)
	}

	if err != nil && err != migrate.ErrNoChange {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	version, dirty, err := m.Version()
	if err != nil {
		// If the error is "no migration", it just means we are at version 0
		if err == migrate.ErrNilVersion {
			slog.Info("✅ migration complete",
				slog.Group("migration",
					slog.Uint64("version", 0),
					slog.Bool("dirty", false),
					slog.String("direction", *direction),
				),
			)
			return // Exit gracefully
		}

		slog.Error("failed to get version", "error", err)
		os.Exit(1)
	}

	slog.Info("✅ migration complete",
		slog.Group("migration",
			slog.Uint64("version", uint64(version)),
			slog.Bool("dirty", dirty),
			slog.String("direction", *direction),
		),
	)
}
