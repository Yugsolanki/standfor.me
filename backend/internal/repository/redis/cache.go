package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
	ttl    time.Duration // default ttl 10 minutes
}

func NewCache(client *redis.Client) *Cache {
	return &Cache{
		client: client,
		ttl:    10 * time.Minute,
	}
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis cache get: %w", err)
	}
	return val, nil
}

func (c *Cache) Set(ctx context.Context, key string, value []byte) error {
	err := c.client.Set(ctx, key, value, c.ttl).Err()
	if err != nil {
		return fmt.Errorf("redis cache set: %w", err)
	}
	return nil
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	err := c.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("redis cache delete: %w", err)
	}
	return nil
}
