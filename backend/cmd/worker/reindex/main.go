package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Yugsolanki/standfor-me/internal/config"
	meilirepository "github.com/Yugsolanki/standfor-me/internal/repository/meilisearch"
	pgrepository "github.com/Yugsolanki/standfor-me/internal/repository/postgres"
	searchservice "github.com/Yugsolanki/standfor-me/internal/service/search"
	"github.com/jmoiron/sqlx"
)

func main() {
	entity := flag.String("entity", "all", "Entity to reindex: movements|users|organizations|all")
	batchSize := flag.Int("batch-size", 50, "Number of IDs per processing batch")
	batchSleep := flag.Int("batch-sleep-ms", 50, "Milliseconds to sleep between batches")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	ctx := context.Background()

	// --- Config ---
	meiliCfg, err := config.LoadSearchConfig()
	if err != nil {
		logger.Error("loading meilisearch config", "error", err)
		os.Exit(1)
	}

	// --- Database ---
	mainCfg, err := config.Load()
	if err != nil {
		logger.Error("loading database config", "error", err)
		os.Exit(1)
	}

	db, err := sqlx.ConnectContext(ctx, "pgx", mainCfg.Database.DSN())
	if err != nil {
		logger.Error("opening database connection", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// ping with retry
	if err := pingWithRetry(ctx, db, 3, 2*time.Second); err != nil {
		logger.Error("database health check failed", "error", err)
		os.Exit(1)
	}

	// --- Meilisearch Client ---
	meiliClient, err := meilirepository.NewClient(&meiliCfg.Meilisearch, logger)
	if err != nil {
		logger.Error("creating meilisearch client", "error", err)
		os.Exit(1)
	}
	if err := meiliClient.Initialize(ctx); err != nil {
		logger.Error("initializing meilisearch indexes", "error", err)
		os.Exit(1)
	}

	// --- Repositories ---
	movementDataRepo := pgrepository.NewMovementIndexingRepository(db)
	userDataRepo := pgrepository.NewUserIndexingRepository(db)
	orgDataRepo := pgrepository.NewOrganizationIndexingRepository(db)

	// --- Service ---
	svc := searchservice.NewService(
		meilirepository.NewMovementRepository(meiliClient),
		meilirepository.NewUserRepository(meiliClient),
		meilirepository.NewOrganizationRepository(meiliClient),
		movementDataRepo,
		userDataRepo,
		orgDataRepo,
		logger,
	)

	// --- Run Reindex ---
	sleep := time.Duration(*batchSleep) * time.Millisecond
	start := time.Now()

	switch *entity {
	case "movements":
		if err := reindexMovements(ctx, logger, movementDataRepo, svc, *batchSize, sleep); err != nil {
			logger.Error("reindexing movements failed", "error", err)
			os.Exit(1)
		}
	case "users":
		if err := reindexUsers(ctx, logger, userDataRepo, svc, *batchSize, sleep); err != nil {
			logger.Error("reindexing users failed", "error", err)
			os.Exit(1)
		}

	case "organizations":
		if err := reindexOrganizations(ctx, logger, orgDataRepo, svc, *batchSize, sleep); err != nil {
			logger.Error("reindexing organizations failed", "error", err)
			os.Exit(1)
		}
	case "all":
		steps := []struct {
			name string
			fn   func() error
		}{
			{
				"movements",
				func() error {
					return reindexMovements(ctx, logger, movementDataRepo, svc, *batchSize, sleep)
				},
			},
			{
				"users",
				func() error {
					return reindexUsers(ctx, logger, userDataRepo, svc, *batchSize, sleep)
				},
			},
			{
				"organizations",
				func() error {
					return reindexOrganizations(ctx, logger, orgDataRepo, svc, *batchSize, sleep)
				},
			},
		}

		for _, step := range steps {
			stepStart := time.Now()
			logger.Info("starting reindex step", "entity", step.name)

			if err := step.fn(); err != nil {
				logger.Error("reindex step failed",
					"entity", step.name,
					"error", err,
				)
				os.Exit(1)
			}

			logger.Info("reindex step complete",
				"entity", step.name,
				"duration", time.Since(stepStart).String(),
			)
		}
	default:
		logger.Error("unknown entity flag value",
			"entity", *entity,
			"allowed", "movements|users|organizations|all",
		)
		os.Exit(1)
	}

	logger.Info("reindex complete", "total_duration", time.Since(start).String())
}

// ------------------------------------------
// Per-entity reindex functions
// ------------------------------------------

// reindexMovements reindexes all movements in the database.
// It uses a cursor-based approach to fetch movements in batches, making it suitable for large datasets.
func reindexMovements(
	ctx context.Context,
	logger *slog.Logger,
	repo *pgrepository.MovementIndexingRepository,
	svc *searchservice.Service,
	batchSize int,
	sleep time.Duration,
) error {
	logger.Info("starting movement reindexing")

	stats := &reindexStats{}

	err := repo.GetAllMovementsForBulkIndex(ctx, batchSize, func(ids []string) error {
		for _, id := range ids {
			if err := svc.IndexMovement(ctx, id); err != nil {
				// Soft failure: log and continue so one bad row doesn't
				// abort a full reindex of thousands of documents.
				logger.Error("failed to index movement",
					"movement_id", id,
					"error", err,
				)
				stats.failed++
				continue
			}
			stats.succeeded++
		}

		logger.Info("movement batch processed",
			"succeeded", stats.succeeded,
			"failed", stats.failed,
		)

		time.Sleep(sleep)
		return nil
	})
	if err != nil {
		return fmt.Errorf("iterating movements for reindex: %w", err)
	}

	logger.Info("movement reindex finished", "succeeded", stats.succeeded, "failed", stats.failed)
	return nil
}

// reindexUsers reindexes all users in the database.
func reindexUsers(
	ctx context.Context,
	logger *slog.Logger,
	repo *pgrepository.UserIndexingRepository,
	svc *searchservice.Service,
	batchSize int,
	sleep time.Duration,
) error {
	logger.Info("starting user reindex")

	stats := &reindexStats{}

	err := repo.GetAllUsersForBulkIndex(ctx, batchSize, func(ids []string) error {
		for _, id := range ids {
			if err := svc.IndexUser(ctx, id); err != nil {
				logger.Error("failed to index user",
					"user_id", id,
					"error", err,
				)
				stats.failed++
				continue
			}
			stats.succeeded++
		}

		logger.Info("user batch processed",
			"succeeded", stats.succeeded,
			"failed", stats.failed,
		)

		time.Sleep(sleep)
		return nil
	})
	if err != nil {
		return fmt.Errorf("iterating users for reindex: %w", err)
	}

	logger.Info("user reindex finished", "succeeded", stats.succeeded, "failed", stats.failed)
	return nil
}

// reindexOrganizations reindexes all organizations in the database.
func reindexOrganizations(
	ctx context.Context,
	logger *slog.Logger,
	repo *pgrepository.OrganizationIndexingRepository,
	svc *searchservice.Service,
	batchSize int,
	sleep time.Duration,
) error {
	logger.Info("starting organization reindex")

	stats := &reindexStats{}

	err := repo.GetAllOrgsForBulkIndex(ctx, batchSize, func(ids []string) error {
		for _, id := range ids {
			if err := svc.IndexOrganization(ctx, id); err != nil {
				logger.Error("failed to index organization",
					"org_id", id,
					"error", err,
				)
				stats.failed++
				continue
			}
			stats.succeeded++
		}

		logger.Info("organization batch processed",
			"succeeded", stats.succeeded,
			"failed", stats.failed,
		)

		time.Sleep(sleep)
		return nil
	})
	if err != nil {
		return fmt.Errorf("iterating organizations for reindex: %w", err)
	}

	logger.Info("organization reindex finished", "succeeded", stats.succeeded, "failed", stats.failed)
	return nil
}

// ------------------------------------------
// Helpers
// ------------------------------------------

// reindexStats tracks per-entity success/failure counts across batches.
// Using a struct (instead of two loose ints) makes it easy to add more
// counters later (e.g., skipped, retried).
type reindexStats struct {
	succeeded int
	failed    int
}

// pingWithRetry attempts to ping the database with exponential backoff
// to handle transient network issues during startup.
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
