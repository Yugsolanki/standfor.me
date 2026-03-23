package redis

import (
	"testing"
	"time"

	"github.com/Yugsolanki/standfor-me/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestNewRedisClient_Fail(t *testing.T) {
	cfg := config.RedisConfig{
		Addr:        "localhost:9999",
		DialTimeout: 100 * time.Millisecond,
	}

	client, err := NewRedisClient(cfg)

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to ping redis")
}
