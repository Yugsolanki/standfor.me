package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Default(t *testing.T) {
	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "development", cfg.Env)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.False(t, cfg.IsProduction())
}

func TestLoad_CustomPort(t *testing.T) {
	t.Setenv("SERVER_PORT", "9090")
	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, cfg.Server.Port, 9090)
}

func TestLoad_ProductionValidation(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "secret")

	cfg, err := Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}
