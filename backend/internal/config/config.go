// internal/config/config.go
package config

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"
	"github.com/joho/godotenv"
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
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
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
	loadedEnv := false
	for _, envPath := range []string{".env", "../.env", "../../.env"} {
		if err := godotenv.Load(envPath); err == nil {
			loadedEnv = true
			break
		}
	}
	if !loadedEnv {
		slog.Warn("no .env file found, falling back to environment variables")
	}

	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	SetDefaults(v)

	decodeHook := mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToTimeHookFunc(time.RFC3339),
	)

	var cfg Config
	if err := v.Unmarshal(&cfg, viper.DecodeHook(decodeHook)); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	validate := validator.New()
	validate.RegisterStructValidation(ExtraStructValidation, Config{})
	if err := validate.Struct(cfg); err != nil {
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

func SetDefaults(v *viper.Viper) {
	// App
	v.SetDefault("app_env", "development")

	// Server
	v.SetDefault("server.host", "localhost")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "10s")
	v.SetDefault("server.write_timeout", "15s")
	v.SetDefault("server.shutdown_timeout", "30s")
	v.SetDefault("server.request_timeout", "30s")

	// Worker
	v.SetDefault("worker.concurrency", 10)

	// Database (PostgreSQL)
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.password", "postgres")
	v.SetDefault("database.dbname", "standfor_dev")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.max_conn_lifetime", "5m")
	v.SetDefault("database.max_conn_idle_time", "1m")
	v.SetDefault("database.health_check_interval", "30s")

	// Redis
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.password", "default-redis-password")
	v.SetDefault("redis.dial_timeout", "5s")
	v.SetDefault("redis.read_timeout", "3s")
	v.SetDefault("redis.write_timeout", "3s")
	v.SetDefault("redis.pool_size", 10)

	// JWT
	v.SetDefault("jwt.secret", "default-jwt-secret")
	v.SetDefault("jwt.access_token_ttl", "15m")
	v.SetDefault("jwt.refresh_token_ttl", "168h") // 7 days
	v.SetDefault("jwt.issuer", "standfor.me")

	// Rate Limiting
	v.SetDefault("rate_limit.global.limit", 100)
	v.SetDefault("rate_limit.global.window", "1m")
}
