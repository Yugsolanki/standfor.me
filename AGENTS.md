# AGENTS.md - Standfor.me Development Guide

This file provides instructions for agentic coding agents operating in this repository.

## Project Overview

Standfor.me is a Golang backend API using chi router, PostgreSQL, and Redis. The frontend is Vue.js (separate repository).

## Build, Lint, and Test Commands

### Backend (Go)

```bash
# Navigate to backend
cd backend

# Run the API server
go run ./cmd/api

# Build the binary
go build -o ./tmp/main ./cmd/api

# Run all tests
go test ./...

# Run a single test (by name)
go test ./internal/middleware/ratelimit -run TestSlidingWindow_BasicRateLimit

# Run tests with verbose output
go test -v ./...

# Run tests with race detector
go test -race ./...

# Run linter (golangci-lint)
golangci-lint run

# Run linter on specific file
golangci-lint run ./internal/config/config.go

# Hot reload during development (requires air)
air
```

### Database Migrations

```bash
# Run migrations
go run ./cmd/migrate
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

### Project Structure

```
backend/
├── cmd/
│   ├── api/          # Main API entrypoint
│   └── migrate/      # Database migrations
├── internal/
│   ├── config/       # Configuration loading
│   ├── domain/       # Domain errors and types
│   ├── middleware/   # HTTP middleware (ratelimit, security, etc.)
│   ├── pkg/          # Shared packages (response, validator, crypto)
│   ├── repository/   # Data access (postgres, redis)
│   └── server/       # HTTP server setup
└── tmp/              # Build output (gitignored)
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
- Run `go mod tidy` after adding dependencies

### Commit Messages

Follow conventional commits:
- `feat: add user authentication`
- `fix: resolve rate limiter memory leak`
- `refactor: extract common validation logic`
- `test: add concurrent request tests`
