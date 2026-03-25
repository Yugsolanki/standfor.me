package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Yugsolanki/standfor-me/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testConfig = config.DatabaseConfig{
	Host:                "localhost",
	Port:                5432,
	User:                "postgres",
	Password:            "wrongpassword",
	DBName:              "postgres",
	SSLMode:             "disable",
	MaxOpenConns:        10,
	MaxIdleConns:        5,
	MaxConnLifetime:     time.Minute,
	MaxConnIdleTime:     time.Minute,
	HealthCheckInterval: 0,
}

func TestNewConnection_Success(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("Skipping integration test: TEST_DATABASE_URL not set")
	}

	cfg := testConfig

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := NewConnection(ctx, cfg)

	require.NoError(t, err)
	require.NotNil(t, db)

	defer db.Close()
	stats := db.Stats()
	assert.Equal(t, 10, stats.MaxOpenConnections)
}

func TestNewConnection_InvalidDSN(t *testing.T) {
	cfg := testConfig

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	db, err := NewConnection(ctx, cfg)

	assert.Error(t, err)
	assert.Nil(t, db)
}
