package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSearchConfig(t *testing.T) {
	cfg, err := LoadSearchConfig()
	require.NoError(t, err)
	assert.NotNil(t, cfg)
}

func TestLoadSearchConfig_CustomFields(t *testing.T) {
	t.Setenv("MEILI_MASTER_KEY", "test-key")
	t.Setenv("MEILI_HOST", "test-host")
	t.Setenv("MEILI_TIMEOUT_MS", "10000")
	t.Setenv("MEILI_MAX_RETRIES", "10")

	cfg, err := LoadSearchConfig()
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "test-key", cfg.Meilisearch.MasterKey)
	assert.Equal(t, "test-host", cfg.Meilisearch.Host)
	assert.Equal(t, 10000, cfg.Meilisearch.TimeoutMs)
	assert.Equal(t, 10, cfg.Meilisearch.MaxRetries)
}
