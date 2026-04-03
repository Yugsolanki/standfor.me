// internal/config/config.go
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type Config struct {
	Env       string          `mapstructure:"app_env" validate:"required,oneof=development production"`
	Server    ServerConfig    `mapstructure:"server" validate:"required"`
	Database  DatabaseConfig  `mapstructure:"database" validate:"required"`
	Redis     RedisConfig     `mapstructure:"redis" validate:"required"`
	JWT       JWTConfig       `mapstructure:"jwt" validate:"required"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit" validate:"required"`
	Worker    WorkerConfig    `mapstructure:"worker" validate:"required"`
}

type ServerConfig struct {
	Host            string        `mapstructure:"host" validate:"required"`
	Port            int           `mapstructure:"port" validate:"required,min=1,max=65535"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout" validate:"required"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout" validate:"required"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout" validate:"required"`
	RequestTimeout  time.Duration `mapstructure:"request_timeout" validate:"required"`
}

type WorkerConfig struct {
	Concurrency int `mapstructure:"concurrency" validate:"required,min=1"`
}

type DatabaseConfig struct {
	Host                string        `mapstructure:"host" validate:"required"`
	Port                int           `mapstructure:"port" validate:"required"`
	User                string        `mapstructure:"user" validate:"required"`
	Password            string        `mapstructure:"password" validate:"required"`
	DBName              string        `mapstructure:"dbname" validate:"required"`
	SSLMode             string        `mapstructure:"sslmode" validate:"required"`
	MaxOpenConns        int           `mapstructure:"max_open_conns" validate:"required,min=1"`
	MaxIdleConns        int           `mapstructure:"max_idle_conns" validate:"required,min=1"`
	MaxConnLifetime     time.Duration `mapstructure:"max_conn_lifetime" validate:"required"`
	MaxConnIdleTime     time.Duration `mapstructure:"max_conn_idle_time" validate:"required"`
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval" validate:"required"`
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.DBName, c.SSLMode)
}

type RedisConfig struct {
	Addr         string        `mapstructure:"addr"  validate:"required"`
	Password     string        `mapstructure:"password"  validate:"required"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"  validate:"required"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"  validate:"required"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"  validate:"required"`
	PoolSize     int           `mapstructure:"pool_size"  validate:"required,min=1"`
}

type JWTConfig struct {
	Secret          string        `mapstructure:"secret"  validate:"required"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"  validate:"required"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"  validate:"required"`
	Issuer          string        `mapstructure:"issuer"  validate:"required"`
}

// --- Rate Limiters ---
type RateLimitGlobalConfig struct {
	Limit  int           `mapstructure:"limit"  validate:"required,min=1"`
	Window time.Duration `mapstructure:"window"  validate:"required"`
}

type RateLimitConfig struct {
	Global RateLimitGlobalConfig `mapstructure:"global"  validate:"required"`
}

func Load() (*Config, error) {
	// --- Setup Viper ---
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("configs")
	v.AddConfigPath("backend/configs")
	v.AddConfigPath("../configs")
	v.AddConfigPath("../../configs")
	v.AddConfigPath("../../../configs")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read base config: %w", err)
	}

	// --- Determine Environment ---
	v.SetEnvPrefix("")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // server.port -> SERVER_PORT
	v.AutomaticEnv()

	env := v.GetString("app_env")
	if env == "" {
		env = "development"
	}

	// --- Merge the environment-specific override ---
	v.SetConfigName(fmt.Sprintf("config.%s", env))

	if err := v.MergeInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to merge %s config: %w", env, err)
		}
		fmt.Printf("warning: no config.%s.yaml found, using base only\n", env)
	}

	// --- Unmarshal into the struct ---
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// --- Validate every field ---
	validate := validator.New()
	validate.RegisterStructValidation(ExtraStructValidation, Config{})
	if err := validate.Struct(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

func ExtraStructValidation(sl validator.StructLevel) {
	cfg := sl.Current().Interface().(Config)

	if cfg.Env == "production" {
		if len(cfg.JWT.Secret) < 32 || cfg.JWT.Secret == "default-jwt-secret" {
			sl.ReportError(cfg.JWT.Secret, "JWT_SECRET", "JWT.Secret", "jwtSecretProd", "JWT secret must be at least 32 characters long and not the default value")
		}
	}
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}
