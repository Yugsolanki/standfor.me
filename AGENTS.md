# AGENTS.md - Standfor.me Development Guide

This file provides instructions for agentic coding agents operating in this repository.

## Project Overview

Standfor.me is a Golang backend API using chi router, PostgreSQL, and Redis. The frontend is Vue.js (separate repository).

## Build, Lint, and Test Commands

The project uses a **Makefile** at the repo root for all common operations. Run `make help` to see all targets.

### Primary Targets

```bash
# Full pipeline: tidy → format → lint → test → build
make all

# Hot-reload during development (requires air)
make run-dev

# Run linter
make lint

# Auto-fix lint issues
make lint-fix

# Format all Go files with gofumpt
make format

# Run all tests
make test

# Run tests with race detector
make test-race

# Generate test coverage report (HTML)
make coverage-html

# Run a single test
make test-one PKG=./internal/middleware/ratelimit NAME=TestSlidingWindow

# Run migrations up / down
make migrate-up
make migrate-down

# Create a new migration file
make migrate-create NAME=create_foo

# Generate Swagger docs
make swag

# Start infrastructure (postgres + redis)
make docker-up-infra

# Stop all containers
make docker-down
```

### All Available Make Targets

| Category | Targets |
|---|---|
| **Build** | `build` (default), `build-api`, `build-migrate`, `build-worker`, `build-all` |
| **Run** | `run`, `run-dev` (hot-reload with air) |
| **Migrations** | `migrate-up`, `migrate-down`, `migrate-status`, `migrate-create NAME=x` |
| **Tests** | `test`, `test-verbose`, `test-race`, `test-coverage`, `coverage-report`, `coverage-text`, `coverage-html`, `test-one PKG=... NAME=...` |
| **Lint & Format** | `lint`, `lint-fix`, `format` |
| **Dependencies** | `tidy`, `deps-update`, `deps-cleanup`, `deps-vuln` (govulncheck) |
| **Docs** | `swag` |
| **Docker** | `docker-up`, `docker-up-infra`, `docker-down`, `docker-logs`, `docker-logs-svc SVC=x`, `docker-clean`, `docker-build` |
| **Release** | `goreleaser-snapshot`, `goreleaser-release` |
| **Tools** | `install-tools` (golangci-lint, swag, gofumpt, air, goreleaser, govulncheck) |
| **Other** | `clean`, `all`, `help` |

### Bare Go Commands (Fallback)

If you prefer not to use Make, the equivalent raw commands are:

```bash
cd backend

# Build
go build -o ./tmp/main ./cmd/api

# Test
go test ./...
go test -v ./internal/middleware/ratelimit -run TestSlidingWindow_BasicRateLimit
go test -race ./...

# Lint
golangci-lint run
golangci-lint run ./internal/config/config.go

# Format
gofumpt -l -w .

# Modules
go mod tidy

# Swagger
swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

# Migrations
go run ./cmd/migrate

# Hot reload
air
```

## Code Style Guidelines

### Imports

Group imports in this order (blank line between groups):
1. Standard library
2. External packages
3. Internal packages

```go
import (
    "context"
    "fmt"
    "log/slog"
    "net/http"
    "time"

    "github.com/Yugsolanki/standfor-me/internal/config"
    "github.com/Yugsolanki/standfor-me/internal/middleware"
    "github.com/go-chi/chi/v5"
    "github.com/redis/go-redis/v9"
)
```

### Formatting

- Use `gofumpt` for formatting (enabled in golangci-lint)
- Run `golangci-lint run --fix` to auto-fix issues
- Maximum line length: 100 characters (soft limit)

### Types and Interfaces

- Use concrete types over interfaces unless necessary
- Define interfaces where they are consumed (e.g., repository interfaces)
- Use `any` instead of `interface{}`

### Naming Conventions

- **Files**: snake_case (e.g., `rate_limit.go`, `config_test.go`)
- **Types/Functions**: PascalCase (e.g., `Server`, `NewServer`)
- **Variables/Methods**: camelCase (e.g., `cfg`, `logger`)
- **Constants**: PascalCase or SCREAMING_SNAKE_CASE for enum-like values
- **Packages**: short, lowercase, no underscores (e.g., `pkg`, `repo`)

### Error Handling

- Use custom domain errors from `internal/domain/errors.go`
- Always handle errors explicitly (no `_` discard)
- Return errors with context using `fmt.Errorf("context: %w", err)`
- Use structured logging for errors with the `error` key

```go
if err != nil {
    return nil, fmt.Errorf("failed to create limiter: %w", err)
}
```

### Structured Logging

- Use `log/slog` for all logging
- Use structured attributes (no printf-style formatting)

```go
slog.Info("server started", "port", cfg.Server.Port)
slog.Error("connection failed", "error", err)
```

### HTTP Handlers

- Use chi router patterns
- Return JSON responses using `response.JSON()` or `response.JSONMessage()`
- Return errors using `response.JSONError()`

```go
func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
    response.JSON(w, r, http.StatusOK, data)
}
```

### Configuration

- Use `viper` + `mapstructure` for configuration
- Define config structs with `mapstructure` tags
- Use `validator` for validation
- Load .env files with `godotenv`

### Testing

- Use standard Go testing with `testify` (assert/require)
- Create helper functions with `t.Helper()`
- Skip tests when external dependencies unavailable
- Use table-driven tests for multiple cases

```go
func TestExample(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid input", "test", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test code
        })
    }
}
```

- Tests requiring Redis should use `testRedisClient(t)` helper which:
  - Uses `REDIS_URL` env var or defaults to `localhost:6379`
  - Uses Redis DB 15 for tests
  - Skips test if Redis unavailable

### Makefile Workflow

Recommended daily workflow:

1. `make docker-up-infra` — Start Postgres and Redis
2. `make run-dev` — Start the API with hot-reload
3. Code your changes
4. `make format` — Format before reviewing
5. `make lint` — Catch issues early
6. `make test-race` — Run tests with race detection
7. `make migrate-create NAME=xxx` — When schema changes are needed
8. `make clean` — Remove build artifacts

Before committing:

```bash
make all   # tidy → format → lint → test → build
```

### Project Structure

```
.
├── Makefile                  # All build, test, lint, docker targets
├── docker-compose.yaml       # Postgres, Redis, pgAdmin, RedisInsight
├── backend/
│   ├── cmd/
│   │   ├── api/              # Main API entrypoint
│   │   ├── migrate/          # Database migration CLI
│   │   └── worker/           # Asynq background worker
│   ├── configs/              # Application config files
│   ├── docs/                 # Swagger-generated API docs
│   ├── migrations/           # SQL migration files (up/down)
│   ├── internal/
│   │   ├── config/           # Configuration loading (viper)
│   │   ├── domain/           # Domain errors and types
│   │   ├── middleware/       # HTTP middleware (ratelimit, security, etc.)
│   │   │   └── ratelimit/   # Redis sliding-window rate limiter
│   │   ├── pkg/              # Shared packages (response, validator, crypto, jwt, pagination)
│   │   ├── repository/       # Data access layer
│   │   │   ├── postgres/    # PostgreSQL repositories
│   │   │   └── redis/       # Redis client & cache
│   │   ├── server/           # Chi router & HTTP server setup
│   │   └── service/          # Business logic (auth, user)
│   ├── tmp/                  # Build output (gitignored)
│   ├── go.mod / go.sum
│   ├── .golangci.yml
│   └── .air.toml
└── .env                      # Environment variables
```

### Middleware Order

When adding middleware to chi router, use this order:
1. Request ID
2. Recoverer
3. Logger
4. Security Headers
5. CORS
6. Timeout
7. Rate Limiting
8. Payload Limiting
9. Compression
10. CSP

### Dependency Management

- Use Go 1.25.5 (per go.mod)
- Run `make tidy` (or `go mod tidy`) after adding dependencies
- Run `make deps-update` to update all direct dependencies to latest minor/patch
- Run `make deps-vuln` to scan for known vulnerabilities (govulncheck)
- Use `make install-tools` to install all dev tools at once

### Commit Messages

Follow conventional commits:
- `feat: add user authentication`
- `fix: resolve rate limiter memory leak`
- `refactor: extract common validation logic`
- `test: add concurrent request tests`
