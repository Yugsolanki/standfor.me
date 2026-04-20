package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type MeilisearchConfig struct {
	Meilisearch MeilisearchRoot `mapstructure:"meilisearch"`
}

type MeilisearchRoot struct {
	Host       string                 `mapstructure:"host"`
	MasterKey  string                 `mapstructure:"master_key"`
	TimeoutMs  int                    `mapstructure:"timeout_ms"`
	MaxRetries int                    `mapstructure:"max_retries"`
	Indexes    map[string]IndexConfig `mapstructure:"indexes"`
}

type IndexConfig struct {
	UID                  string              `mapstructure:"uid"`
	PrimaryKey           string              `mapstructure:"primary_key"`
	SearchableAttributes []string            `mapstructure:"searchable_attributes"`
	FilterableAttributes []string            `mapstructure:"filterable_attributes"`
	SortableAttributes   []string            `mapstructure:"sortable_attributes"`
	RankingRules         []string            `mapstructure:"ranking_rules"`
	TypoTolerance        TypoToleranceConfig `mapstructure:"typo_tolerance"`
	Pagination           PaginationConfig    `mapstructure:"pagination"`
}

type TypoToleranceConfig struct {
	Enabled             bool                `mapstructure:"enabled"`
	MinWordSizeForTypos MinWordSizeForTypos `mapstructure:"min_word_size_for_typos"`
}

type MinWordSizeForTypos struct {
	OneTypo  int64 `mapstructure:"one_typo"`
	TwoTypos int64 `mapstructure:"two_typos"`
}

type PaginationConfig struct {
	MaxTotalHits int64 `mapstructure:"max_total_hits"`
}

func LoadSearchConfig() (*MeilisearchConfig, error) {
	v := viper.New()
	v.SetConfigName("search")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("configs")
	v.AddConfigPath("backend/configs")
	v.AddConfigPath("../configs")
	v.AddConfigPath("../../configs")
	v.AddConfigPath("../../../configs")
	v.AddConfigPath("../../../../configs")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read base search config: %w", err)
	}

	// --- Determine Environment ---
	v.SetEnvPrefix("")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // server.port -> SERVER_PORT
	v.AutomaticEnv()

	// Bind env variables
	_ = v.BindEnv("meilisearch.host", "MEILI_HOST")
	_ = v.BindEnv("meilisearch.timeout_ms", "MEILI_TIMEOUT_MS")
	_ = v.BindEnv("meilisearch.max_retries", "MEILI_MAX_RETRIES")
	_ = v.BindEnv("meilisearch.master_key", "MEILI_MASTER_KEY")

	var cfg MeilisearchConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal search config: %w", err)
	}

	if err := validateMeilisearchConfig(&cfg.Meilisearch); err != nil {
		return nil, fmt.Errorf("failed to validate search config: %w", err)
	}

	return &cfg, nil
}

func validateMeilisearchConfig(cfg *MeilisearchRoot) error {
	if cfg.Host == "" {
		return fmt.Errorf("meilisearch host is required")
	}
	if cfg.MasterKey == "" {
		return fmt.Errorf("meilisearch master key is required")
	}
	if cfg.TimeoutMs <= 0 {
		cfg.TimeoutMs = 2000
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if len(cfg.Indexes) == 0 {
		return fmt.Errorf("meilisearch indexes is required")
	}
	return nil
}
