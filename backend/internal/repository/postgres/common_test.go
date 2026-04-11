package postgres

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jmoiron/sqlx"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testDB    *sqlx.DB
	dbInitErr error
	once      sync.Once
)

func getTestDB(t *testing.T) *sqlx.DB {
	once.Do(func() {
		ctx := context.Background()

		// Spin up the postgres container
		pgContainer, err := postgres.Run(ctx,
			"postgres:17-alpine",
			postgres.WithDatabase("test_db"),
			postgres.WithUsername("user"),
			postgres.WithPassword("password"),
			postgres.WithSQLDriver("pgx"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(10*time.Second),
			),
		)
		if err != nil {
			dbInitErr = fmt.Errorf("failed to start postgres container: %v", err)
			return
		}

		// Get dynamically assigned connection string
		connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			dbInitErr = fmt.Errorf("failed to get connection string: %v", err)
			return
		}

		// Connect using sqlx
		db, err := sqlx.Connect("pgx", connStr)
		if err != nil {
			dbInitErr = fmt.Errorf("failed to connect to database: %v", err)
			return
		}

		// Verify the connection
		if err := db.Ping(); err != nil {
			dbInitErr = fmt.Errorf("failed to ping database: %v", err)
			return
		}

		// Migrate the database
		m, err := migrate.New("file://../../../migrations", connStr)
		if err != nil {
			dbInitErr = fmt.Errorf("failed to create migrate instance: %v", err)
			return
		}
		defer m.Close()

		if err := m.Up(); err != nil {
			dbInitErr = fmt.Errorf("failed to migrate database: %v", err)
			return
		}

		testDB = db
	})

	if dbInitErr != nil {
		t.Skipf("Database not available: %v", dbInitErr)
	}
	return testDB
}
