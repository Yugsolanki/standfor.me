package meilisearch

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Yugsolanki/standfor-me/internal/config"
	"github.com/meilisearch/meilisearch-go"
)

type Client struct {
	ms     meilisearch.ServiceManager
	cfg    *config.MeilisearchRoot
	logger *slog.Logger

	MovementsIndex     meilisearch.IndexManager
	UsersIndex         meilisearch.IndexManager
	OrganizationsIndex meilisearch.IndexManager
}

func NewClient(cfg *config.MeilisearchRoot, logger *slog.Logger) (*Client, error) {
	httpClient := &http.Client{
		Timeout: time.Duration(cfg.TimeoutMs) * time.Second,
	}

	ms := meilisearch.New(cfg.Host, meilisearch.WithAPIKey(cfg.MasterKey), meilisearch.WithCustomClient(httpClient))

	c := &Client{
		ms:     ms,
		cfg:    cfg,
		logger: logger,
	}

	return c, nil
}

// Initialize creates the indexes and applies the settings from config.
// It uses the ensureIndex helper to create and configure each index.
func (c *Client) Initialize(ctx context.Context) error {
	indexNames := []string{"movements", "users", "organizations"}
	for _, name := range indexNames {
		idxCfg, ok := c.cfg.Indexes[name]
		if !ok {
			return fmt.Errorf("meilisearch: no config found for index %q", name)
		}

		index, err := c.ensureIndex(ctx, idxCfg)
		if err != nil {
			return fmt.Errorf("ensuring index %q: %w", name, err)
		}

		switch name {
		case "movements":
			c.MovementsIndex = index
		case "users":
			c.UsersIndex = index
		case "organizations":
			c.OrganizationsIndex = index
		}

		c.logger.InfoContext(ctx, "search index ready", slog.String("index", name))
	}
	return nil
}

// ensureIndex creates the index if it does not exist, then applies the
// full settings from config. Meilisearch's update settings API is idempotent.
func (c *Client) ensureIndex(ctx context.Context, idxCfg config.IndexConfig) (meilisearch.IndexManager, error) {
	// CreateIndex is a no-op if the index already exists.
	task, err := c.ms.CreateIndex(&meilisearch.IndexConfig{
		Uid:        idxCfg.UID,
		PrimaryKey: idxCfg.PrimaryKey,
	})
	if err != nil {
		// Meilisearch returns an error if index already exists with a
		// specific error code; we treat it as non-fatal.
		c.logger.DebugContext(ctx, "create index (may already exist)",
			slog.String("index", idxCfg.UID),
			slog.Any("error", err),
		)
	} else {
		// Wait for the creation task to complete before applying settings.
		if _, err := c.ms.WaitForTaskWithContext(ctx, task.TaskUID, 100*time.Millisecond); err != nil {
			return nil, fmt.Errorf("waiting for index creation task: %w", err)
		}
	}

	index := c.ms.Index(idxCfg.UID)

	if err := c.applySettings(ctx, index, idxCfg); err != nil {
		return nil, fmt.Errorf("applying settings to index %q: %w", idxCfg.UID, err)
	}

	return index, nil
}

// applySettings pushes the full settings from config to the Meilisearch index.
// This is called at startup to ensure the index always matches the config file.
func (c *Client) applySettings(ctx context.Context, index meilisearch.IndexManager, idxCfg config.IndexConfig) error {
	settings := &meilisearch.Settings{
		SearchableAttributes: idxCfg.SearchableAttributes,
		FilterableAttributes: idxCfg.FilterableAttributes,
		SortableAttributes:   idxCfg.SortableAttributes,
		RankingRules:         idxCfg.RankingRules,
		TypoTolerance: &meilisearch.TypoTolerance{
			Enabled: idxCfg.TypoTolerance.Enabled,
			MinWordSizeForTypos: meilisearch.MinWordSizeForTypos{
				OneTypo:  idxCfg.TypoTolerance.MinWordSizeForTypos.OneTypo,
				TwoTypos: idxCfg.TypoTolerance.MinWordSizeForTypos.TwoTypos,
			},
		},
		Pagination: &meilisearch.Pagination{
			MaxTotalHits: idxCfg.Pagination.MaxTotalHits,
		},
	}

	task, err := index.UpdateSettings(settings)
	if err != nil {
		return fmt.Errorf("updating settings: %w", err)
	}

	result, err := c.ms.WaitForTaskWithContext(ctx, task.TaskUID, 200*time.Millisecond)
	if err != nil {
		return fmt.Errorf("waiting for settings task: %w", err)
	}

	if result.Status == meilisearch.TaskStatusFailed {
		return fmt.Errorf("settings task failed: %s", result.Error.Message)
	}

	c.logger.InfoContext(ctx, "index settings applied",
		slog.String("index", idxCfg.UID),
		slog.Int64("task_uid", result.UID),
	)
	return nil
}

// HealthCheck verifies that the Meilisearch server is reachable.
func (c *Client) HealthCheck() error {
	health, err := c.ms.Health()
	if err != nil {
		return fmt.Errorf("meilisearch health check failed: %w", err)
	}
	if health.Status != "available" {
		return fmt.Errorf("meilisearch status is %q, expected 'available'", health.Status)
	}
	return nil
}
