# AGENTS.md - Standfor.me Development Guide

This document provides comprehensive guidance for AI coding agents working on the Standfor.me codebase. It covers project architecture, development workflows, coding conventions, and agent-specific guidelines.

---

## Table of Contents

1. [Project Overview](#1-project-overview)
2. [Repository Layout](#2-repository-layout)
3. [Backend Architecture](#3-backend-architecture)
4. [Infrastructure & Services](#4-infrastructure--services)
5. [Development Workflow](#5-development-workflow)
6. [Configuration](#6-configuration)
7. [Testing](#7-testing)
8. [API Documentation](#8-api-documentation)
9. [Coding Conventions & Agent Guidelines](#9-coding-conventions--agent-guidelines)
10. [Roadmap Context](#10-roadmap-context)

---

## 1. Project Overview

### What is Standfor.me?

Standfor.me is a digital platform that transforms passive sympathy into **visible, verified advocacy**. It provides users with a centralized public profile where they can declare the social movements and causes they support, with a structured verification system that quantifies the depth of their commitment.

### The Problem

Digital activism suffers from a "credibility gap":
- Support for causes is fragmented across social media platforms
- Performative activism ("slacktivism") is indistinguishable from genuine commitment
- No standardized way to aggregate or verify advocacy across multiple dimensions

### The Solution

Standfor.me creates an "advocacy portfolio" — a shareable public profile (`username.standfor.me`) that displays:
- Movements a user supports
- **Verification badges** proving depth of engagement
- A reputation system based on demonstrated commitment

### Verification Hierarchy

The platform uses a six-tier badge system to validate different levels of engagement:

| Tier | Name | Validation Method |
|------|------|-------------------|
| 0 | **Self-Declared** | User publicly lists causes they align with |
| 1 | **Bronze (Social Proof)** | Digital footprint: following orgs, using advocacy hashtags |
| 2 | **Silver (Financial Proof)** | Charitable contributions, Patreon, donation receipts |
| 3 | **Gold (Action Proof)** | Real-world engagement: rally attendance, volunteer hours, event check-ins |
| 4 | **Platinum (Organization Vouched)** | Verified NGOs officially vouch for user as volunteer/ambassador/staff |
| 5 | **Diamond** | Reserved for exceptional long-term commitment (future) |

This hierarchy creates a **trust layer for digital activism**, allowing users to build a "resume for activism."

---

## 2. Repository Layout

```
standfor.me/
├── AGENTS.md                 # This file - comprehensive agent guide
├── README.md                 # Project overview and quick start
├── Makefile                  # All build, test, lint, docker targets
├── docker-compose.yaml       # Infrastructure stack (Postgres, Redis, Meilisearch, pgAdmin, RedisInsight)
├── .env                      # Environment variables (database, redis, meilisearch secrets)
├── .envrc                    # direnv configuration
├── .goreleaser.yaml          # GoReleaser configuration for releases
├── .gitignore
├── skills-lock.json          # Agent skills configuration
│
├── backend/                  # Go backend API (chi router + PostgreSQL + Redis + Meilisearch)
│   ├── cmd/                  # Application entrypoints
│   │   ├── api/              # Main HTTP API server
│   │   ├── migrate/          # Database migration CLI
│   │   ├── seed/             # Data seeding utility
│   │   └── worker/           # Asynq background worker
│   │       └── reindex/      # Meilisearch reindexing worker
│   ├── configs/              # Configuration files
│   │   ├── config.yaml       # Base configuration
│   │   ├── config.development.yaml
│   │   ├── config.production.yaml
│   │   └── search.yaml       # Meilisearch index configurations
│   ├── docs/                 # Swagger-generated API documentation
│   │   ├── docs.go
│   │   ├── swagger.json
│   │   └── swagger.yaml
│   ├── migrations/           # SQL migration files (up/down pairs)
│   ├── internal/             # Private application code
│   │   ├── config/           # Configuration loading (viper)
│   │   ├── domain/           # Domain types and errors
│   │   │   ├── errors.go
│   │   │   ├── user.go
│   │   │   ├── organization.go
│   │   │   ├── movement.go
│   │   │   ├── category.go
│   │   │   ├── movement_category.go
│   │   │   ├── user_movement.go
│   │   │   ├── token.go
│   │   │   └── search/       # Search-specific domain types
│   │   ├── middleware/       # HTTP middleware stack
│   │   │   ├── auth.go            # JWT authentication
│   │   │   ├── rateLimit/         # Redis sliding-window rate limiter
│   │   │   ├── cors.go
│   │   │   ├── csp_nonce.go
│   │   │   ├── security_headers.go
│   │   │   ├── compress.go
│   │   │   ├── timeout.go
│   │   │   ├── payload_limit.go
│   │   │   ├── recoverer.go
│   │   │   ├── request_id.go
│   │   │   └── canonical_logger.go
│   │   ├── pkg/              # Shared utilities
│   │   │   ├── crypto/       # Password hashing (bcrypt)
│   │   │   ├── jwt/          # JWT issuance and validation
│   │   │   ├── logger/       # Structured logging helpers
│   │   │   ├── pagination/   # Pagination utilities
│   │   │   ├── requestid/    # Request ID context helpers
│   │   │   ├── response/     # HTTP response helpers
│   │   │   ├── validator/    # Custom validator (go-playground/validator)
│   │   │   └── httputil/     # HTTP utilities
│   │   ├── repository/       # Data access layer
│   │   │   ├── postgres/     # PostgreSQL repositories
│   │   │   │   ├── connection.go
│   │   │   │   ├── user_repository.go
│   │   │   │   ├── movement_repository.go
│   │   │   │   ├── movement_indexing.go
│   │   │   │   ├── user_indexing.go
│   │   │   │   ├── organization_indexing.go
│   │   │   │   └── refresh_token_repository.go
│   │   │   ├── redis/        # Redis client and caching
│   │   │   └── meilisearch/  # Meilisearch search repositories
│   │   │       ├── client.go
│   │   │       ├── filter_builder.go
│   │   │       ├── user_repository.go
│   │   │       ├── movement_repository.go
│   │   │       └── organization_repository.go
│   │   ├── server/           # Chi router and HTTP handlers
│   │   │   ├── server.go     # Server setup and middleware
│   │   │   ├── routes.go     # Route definitions
│   │   │   ├── handler.go    # Base handler utilities
│   │   │   ├── auth_handler.go
│   │   │   ├── user_handler.go
│   │   │   ├── movement_handler.go
│   │   │   ├── search_handler.go
│   │   │   └── ratelimit.go
│   │   └── service/          # Business logic layer
│   │       ├── auth_service.go
│   │       ├── user_service.go
│   │       ├── movement_service.go
│   │       └── search/
│   │           ├── service.go       # Search orchestration
│   │           ├── interfaces.go    # Repository interfaces
│   │           └── index_data.go    # Indexing data structures
│   ├── tmp/                  # Build output (gitignored)
│   ├── go.mod / go.sum
│   ├── .golangci.yml         # Linter configuration
│   └── .air.toml             # Hot-reload configuration
│
├── storage/                  # Persistent volumes (gitignored)
│   ├── postgres/             # PostgreSQL data
│   ├── redis/                # Redis data
│   └── meilisearch/          # Meilisearch index data
│
└── cmd/                      # Root-level command scripts (if any)
```

---

## 3. Backend Architecture

### Layered Architecture

The backend follows a **clean architecture** pattern with clear separation of concerns:

```
HTTP Request Flow:
  Client → Middleware Stack → Handler → Service → Repository → Database/Search

Response Flow:
  Database/Search → Repository → Service → Handler → Middleware → Client
```

#### Layer Responsibilities

| Layer | Package | Responsibility |
|-------|---------|----------------|
| **Entrypoint** | `cmd/api`, `cmd/worker`, `cmd/migrate` | Application bootstrap, dependency injection |
| **HTTP Handlers** | `internal/server/` | Request parsing, validation, response formatting |
| **Business Logic** | `internal/service/` | Domain logic, orchestration, transaction coordination |
| **Data Access** | `internal/repository/` | SQL queries, Meilisearch operations, Redis caching |
| **Domain Types** | `internal/domain/` | Core entities, value objects, domain errors |
| **Middleware** | `internal/middleware/` | Cross-cutting concerns (auth, rate limiting, logging) |
| **Utilities** | `internal/pkg/` | Shared helpers (JWT, crypto, pagination, response) |

---

### Command Entrypoints (`backend/cmd/`)

#### `cmd/api/main.go` — HTTP API Server

The main application entrypoint. Responsibilities:
- Load configuration from `configs/` directory
- Establish database connections (Postgres, Redis, Meilisearch)
- Initialize repositories, services, and middleware
- Configure chi router with middleware stack
- Start HTTP server with graceful shutdown

**When to use:** Running the API locally or in production.

```bash
make run-dev          # Hot-reload during development
make build-api && ./backend/tmp/standfor-me  # Production binary
```

#### `cmd/migrate/main.go` — Database Migrations

CLI tool for running database migrations using `golang-migrate/migrate`.

**Commands:**
```bash
make migrate-up       # Apply all pending migrations
make migrate-down     # Rollback last migration
make migrate-create NAME=create_foo  # Create new migration pair
```

**When to use:** Schema changes, initial setup, CI/CD deployment.

#### `cmd/seed/main.go` — Data Seeding

Populates the database with initial/fake data for development and testing.

**When to use:** Fresh development environment setup, integration test data.

#### `cmd/worker/main.go` — Background Worker

Runs Asynq-based background jobs for async processing.

**When to use:** Email sending, notification processing, scheduled tasks.

#### `cmd/worker/reindex/main.go` — Meilisearch Reindexer

Dedicated worker for rebuilding Meilisearch indexes from PostgreSQL.

**When to use:** 
- Initial index population
- Full reindex after schema changes
- Recovery from index corruption

```bash
cd backend && go run ./cmd/worker/reindex
```

---

### Domain Types (`internal/domain/`)

Core entities that model the business domain:

#### `user.go` — User Entity

```go
type User struct {
    ID                uuid.UUID
    Username          string     // Unique, for profile URLs
    Email             string
    EmailVerifiedAt   *time.Time
    PasswordHash      *string
    DisplayName       string
    Bio               *string
    AvatarURL         *string
    Location          *string
    ProfileVisibility string     // "public", "private", "unlisted"
    EmbedEnabled      bool
    Role              string     // "user", "moderator", "admin", "superadmin"
    Status            string     // "active", "suspended", "banned", "deactivated"
    CreatedAt         time.Time
    UpdatedAt         time.Time
}
```

**Key Constants:**
- `ProfileVisibilityPublic`, `ProfileVisibilityPrivate`, `ProfileVisibilityUnlisted`
- `RoleUser`, `RoleModerator`, `RoleAdmin`, `RoleSuperAdmin`
- `StatusActive`, `StatusSuspended`, `StatusBanned`, `StatusDeactivated`

#### `organization.go` — Organization Entity

Represents NGOs, non-profits, and advocacy groups that can vouch for users.

```go
type Organization struct {
    ID               uuid.UUID
    Name             string
    Slug             string
    ShortDescription string
    LongDescription  string
    LogoURL          string
    CoverImageURL    string
    WebsiteURL       string
    ContactEmail     string
    EINTaxIDHash     string       // Hashed for privacy
    CountryCode      string
    Status           OrganizationStatus
    VerificationStatus VerificationStatus
    IsVerified       bool
    VerifiedAt       *time.Time
    SocialLinks      SocialLinks  // X, Bluesky, Instagram, etc.
    CreatedByUserID  uuid.UUID
    CreatedAt        time.Time
    UpdatedAt        time.Time
}
```

#### `movement.go` — Movement Entity

Represents social causes and movements users can support.

```go
type Movement struct {
    ID              uuid.UUID
    Slug            string
    Name            string
    ShortDescription string
    LongDescription *string
    ImageURL        *string
    IconURL         *string
    WebsiteURL      *string
    SupporterCount  int
    TrendingScore   float64
    Status          string  // "draft", "active", "archived", "rejected", "pending_review"
    ClaimedByOrgID  *uuid.UUID
    CreatedByUserID *uuid.UUID
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

#### `errors.go` — Domain Errors

Standardized error types for consistent error handling:

```go
// Sentinel errors
var (
    ErrNotFound           = errors.New("resource not found")
    ErrConflict           = errors.New("resource already exists")
    ErrUnauthorized       = errors.New("unauthorized")
    ErrForbidden          = errors.New("forbidden")
    ErrInternal           = errors.New("internal server error")
    ErrValidation         = errors.New("validation failed")
    ErrRateLimit          = errors.New("rate limit exceeded")
    ErrInvalidCredentials = errors.New("invalid credentials")
)

// AppError wrapper for structured errors
type AppError struct {
    Op      string
    Message string
    Details map[string]string
    Cause   error
}
```

**Constructor helpers:**
- `NewNotFoundError(op, message)`
- `NewValidationError(op, details)`
- `NewUnauthorizedError(op, message)`
- `NewConflictError(op, message)`
- `NewInternalError(op, cause)`

---

### Middleware Stack

Middleware is applied in `internal/server/server.go` in this order:

```go
s.router.Use(middleware.RequestID)                    // 1. Generate request ID
s.router.Use(middleware.Recoverer(s.logger))          // 2. Panic recovery
s.router.Use(s.globalRateLimiter())                   // 3. Global rate limiting
s.router.Use(middleware.PayloadLimit(...))            // 4. Max body size
s.router.Use(middleware.Compress)                     // 5. gzip compression
s.router.Use(middleware.CanonicalLogger(s.logger))    // 6. Structured logging
s.router.Use(middleware.CORS(...))                    // 7. CORS headers
s.router.Use(middleware.SecurityHeaders(...))         // 8. Security headers
s.router.Use(middleware.CSPNonce())                   // 9. CSP nonce
s.router.Use(middleware.Timeout(cfg.RequestTimeout))  // 10. Request timeout
```

#### Key Middleware

| Middleware | File | Purpose |
|------------|------|---------|
| `RequestID` | `request_id.go` | Adds `X-Request-ID` header, stores in context |
| `Recoverer` | `recoverer.go` | Catches panics, logs stack trace, returns 500 |
| `globalRateLimiter` | `ratelimit/` | Redis sliding-window rate limiter |
| `PayloadLimit` | `payload_limit.go` | Rejects requests > 1MB by default |
| `CanonicalLogger` | `canonical_logger.go` | Logs all requests with timing and status |
| `CORS` | `cors.go` | Cross-origin request handling |
| `SecurityHeaders` | `security_headers.go` | HSTS, X-Frame-Options, X-Content-Type-Options |
| `CSPNonce` | `csp_nonce.go` | Content-Security-Policy with nonce |
| `Timeout` | `timeout.go` | Request timeout with context cancellation |
| `Authenticate` | `auth.go` | JWT validation, stores claims in context |
| `RequireAuth` | `auth.go` | Alias for Authenticate |
| `RequireRole` | `auth.go` | Role-based access control |
| `RequireMinRole` | `auth.go` | Minimum role level check |

---

### Utilities (`internal/pkg/`)

#### `jwt/` — JWT Service

Handles access token issuance and validation.

```go
type Service struct {
    secret          []byte
    accessTokenTTL  time.Duration  // 15m
    refreshTokenTTL time.Duration  // 168h (7 days)
    issuer          string         // "standfor.me"
}

func (s *Service) IssueAccessToken(user *domain.User) (string, error)
func (s *Service) ValidateAccessToken(raw string) (*domain.AccessTokenClaims, error)
```

**Claims structure:**
```go
type AccessTokenClaims struct {
    UserID uuid.UUID
    Role   string
    Email  string
}
```

#### `crypto/` — Password Hashing

Uses bcrypt for password hashing.

```go
func HashPassword(password string) (string, error)
func VerifyPassword(hash, password string) bool
```

#### `response/` — HTTP Response Helpers

Standardized JSON response formatting.

```go
// Success responses
response.JSON(w, r, http.StatusOK, data)           // With data payload
response.JSONMessage(w, r, http.StatusOK, "OK")    // Message only

// Error responses (handlers should ONLY use this for errors)
response.JSONError(w, r, err)
```

**Response envelope:**
```json
{
  "success": true,
  "data": { ... },
  "message": "..."
}
```

**Error envelope:**
```json
{
  "success": false,
  "error": {
    "message": "...",
    "code": "not_found",
    "details": { "field": "error" }
  },
  "request_id": "..."
}
```

#### `validator/` — Request Validation

Wraps `go-playground/validator` with custom rules.

```go
type Validator struct {
    validate *validator.Validate
}

func (v *Validator) Validate(data any) map[string]string
```

**Custom validators:**
- `username`: Alphanumeric, underscores, hyphens only
- Standard: `required`, `email`, `min`, `max`, `url`, `uuid`, `oneof`

#### `pagination/` — Pagination Helpers

Utilities for offset-based pagination.

#### `requestid/` — Request ID Context

```go
func GetRequestID(ctx context.Context) string
```

#### `logger/` — Logging Helpers

Structured logging utilities.

---

### Search Architecture

Standfor.me uses **Meilisearch** for full-text search across movements, users, and organizations.

#### Indexes (`configs/search.yaml`)

Three indexes are configured:

| Index | UID | Purpose |
|-------|-----|---------|
| `movements` | `movements` | Search movements by name, description, category |
| `users` | `users` | Search advocates by display name, username, bio |
| `organizations` | `organizations` | Search organizations by name, description |

#### Filterable Attributes

Movements support filtering by depth-of-commitment metrics:
- `avg_verification_tier`, `min_verification_tier`, `max_verification_tier`
- `min_badge_level_numeric`, `max_badge_level_numeric`
- `supporter_count`, `trending_score`
- `has_verified_org`, `claimed_by_org_id`
- `category_ids`, `category_slugs`

#### `filter_builder.go` — Query Builder

Fluent builder for constructing Meilisearch filter expressions:

```go
filter := newFilterBuilder().
    addEquals("status", "active").
    addGTEFloat("avg_verification_tier", 2.0).
    addOptionalGTEInt64("supporter_count", req.MinSupporters).
    build()
// Result: "(status = \"active\") AND (avg_verification_tier >= 2.0) AND (supporter_count >= 100)"
```

#### Search Service (`internal/service/search/`)

Orchestrates search operations:

```go
type Service struct {
    movements     MovementSearchRepo
    users         UserSearchRepo
    organizations OrganizationSearchRepo
    
    // Data repositories (Postgres) for indexing
    movementData MovementDataRepo
    userData     UserDataRepo
    orgData      OrgDataRepo
}

// Search methods
func (s *Service) SearchMovements(ctx, req) (*SearchResult[MovementDocument], error)
func (s *Service) SearchUsers(ctx, req) (*SearchResult[UserDocument], error)
func (s *Service) SearchOrganizations(ctx, req) (*SearchResult[OrganizationDocument], error)

// Indexing methods
func (s *Service) IndexMovement(ctx, movementID string) error
func (s *Service) IndexUser(ctx, userID string) error
func (s *Service) IndexOrganization(ctx, orgID string) error
```

#### Indexing Pipeline

1. **Real-time indexing:** After create/update/delete in Postgres, call `IndexX()` to sync to Meilisearch
2. **Batch reindexing:** Run `cmd/worker/reindex` to rebuild entire index from Postgres
3. **Data transformation:** `buildMovementDocument()`, `buildUserDocument()`, `buildOrgDocument()` flatten PostgreSQL data into search documents

---

### Database Migrations (`backend/migrations/`)

Migrations use `golang-migrate/migrate` with sequential numbering.

**Current migrations:**

| File | Purpose |
|------|---------|
| `000001_bootstrap.up.sql` | Initial schema setup |
| `000002_create_users.up.sql` | Users table |
| `000003_create_organizations.up.sql` | Organizations table |
| `000004_create_categories.up.sql` | Categories table |
| `000005_create_refresh_tokens.up.sql` | Refresh tokens table |
| `000006_create_reserved_usernames.up.sql` | Reserved usernames table |
| `000007_create_movements.up.sql` | Movements table |
| `000008_create_movement_categories.up.sql` | Movement categories junction |
| `000009_create_user_movements.up.sql` | User movements junction |

**Migration commands:**
```bash
make migrate-up                    # Apply all pending
make migrate-down                  # Rollback last
make migrate-create NAME=add_foo   # Create 000010_add_foo.up.sql / .down.sql
```

---

## 4. Infrastructure & Services

### Docker Compose Stack (`docker-compose.yaml`)

| Service | Image | Port | Purpose |
|---------|-------|------|---------|
| `postgres` | `postgres:17-alpine` | 5432 | Primary database |
| `pgadmin` | `dpage/pgadmin4` | 5050 | Database administration UI |
| `redis` | `redis:8-alpine` | 6379 | Caching, rate limiting, sessions |
| `redisinsight` | `redis/redisinsight:latest` | 5540 | Redis visualization |
| `meilisearch` | `getmeili/meilisearch:latest` | 7700 | Full-text search engine |

**Network:** All services connected via `standfor-net` bridge network.

**Volumes:**
- `./storage/postgres` → PostgreSQL data
- `./storage/redis` → Redis data
- `./storage/meilisearch` → Meilisearch index

### Starting Infrastructure

```bash
# Start all services
make docker-up

# Start only core infrastructure (postgres + redis)
make docker-up-infra

# View logs
make docker-logs
make docker-logs-svc SVC=postgres

# Stop all
make docker-down

# Clean volumes
make docker-clean
```

### Configuration Files (`backend/configs/`)

#### `config.yaml` — Base Configuration

```yaml
app_env: development

server:
  host: localhost
  port: 8080
  read_timeout: 10s
  write_timeout: 15s
  shutdown_timeout: 30s
  request_timeout: 30s

database:
  host: localhost
  port: 5432
  user: postgres
  dbname: standfor_dev
  sslmode: disable
  max_open_conns: 10
  max_idle_conns: 5

redis:
  addr: localhost:6379
  pool_size: 10

jwt:
  access_token_ttl: 15m
  refresh_token_ttl: 168h
  issuer: standfor.me

rate_limit:
  global:
    limit: 100
    window: 1m
```

#### `config.development.yaml` / `config.production.yaml` — Environment Overrides

Environment-specific overrides merged on top of base config.

#### `search.yaml` — Meilisearch Configuration

Defines index settings (searchable attributes, filterable attributes, ranking rules, typo tolerance).

---

## 5. Development Workflow

### Prerequisites

- **Go** 1.25.5+
- **Docker** & **Docker Compose**
- **Make**

### Quick Start

```bash
# 1. Install development tools
make install-tools

# 2. Start infrastructure
make docker-up-infra

# 3. Run migrations
make migrate-up

# 4. Seed test data (optional)
cd backend && go run ./cmd/seed

# 5. Start API with hot-reload
make run-dev
```

The API will be available at `http://localhost:8080`.

### Common Make Targets

| Target | Description |
|--------|-------------|
| `make run-dev` | Hot-reload dev server (air) |
| `make build` | Build API binary |
| `make build-all` | Build API + migrate + worker binaries |
| `make lint` | Run golangci-lint |
| `make lint-fix` | Auto-fix lint issues |
| `make format` | Format with gofumpt |
| `make test` | Run all tests |
| `make test-race` | Run tests with race detector |
| `make test-one PKG=... NAME=...` | Run single test |
| `make coverage-html` | Generate and open HTML coverage report |
| `make swag` | Regenerate Swagger docs |
| `make migrate-up` | Apply migrations |
| `make migrate-create NAME=x` | Create new migration |
| `make tidy` | Run `go mod tidy` |
| `make all` | Full pipeline: tidy → format → lint → test → build |

### Development Loop

1. **Code** your changes
2. **Hot-reload** automatically restarts the server (`make run-dev`)
3. **Test** with `make test` or `make test-one`
4. **Format** with `make format`
5. **Lint** with `make lint` or `make lint-fix`
6. **Commit** (pre-commit hooks run automatically)

### Reindexing Meilisearch

```bash
# Full reindex
cd backend && go run ./cmd/worker/reindex

# Or use make (if target exists)
make reindex
```

---

## 6. Configuration

### Configuration Layering

Configuration is loaded in this order (later overrides earlier):

1. `configs/config.yaml` — Base defaults
2. `configs/config.development.yaml` — Development overrides (when `APP_ENV=development`)
3. `configs/config.production.yaml` — Production overrides (when `APP_ENV=production`)
4. Environment variables (highest priority)

### Key Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `APP_ENV` | Environment name | `development`, `production` |
| `DATABASE_USER` | PostgreSQL username | `postgres` |
| `DATABASE_PASSWORD` | PostgreSQL password | (see `.env`) |
| `DATABASE_DBNAME` | Database name | `standfor_dev` |
| `DATABASE_PORT` | PostgreSQL port | `5432` |
| `REDIS_PASSWORD` | Redis password | (see `.env`) |
| `REDIS_PORT` | Redis port | `6379` |
| `MEILI_MASTER_KEY` | Meilisearch master key | (see `.env`) |
| `MEILI_HOST` | Meilisearch URL | `http://localhost:7700` |

### Loading Configuration in Code

```go
import "github.com/Yugsolanki/standfor-me/internal/config"

cfg, err := config.Load()
if err != nil {
    return err
}

// Access via cfg.Server, cfg.Database, cfg.Redis, cfg.JWT, etc.
```

---

## 7. Testing

### Test Location

Tests live alongside source files with `_test.go` suffix:
- `internal/server/auth_handler_test.go`
- `internal/service/auth_service_test.go`
- `internal/middleware/auth_test.go`

### Test Patterns

#### Shared Database Setup (`internal/repository/postgres/common_test.go`)

Uses **testcontainers-go** to spin up ephemeral PostgreSQL containers for integration tests:

```go
func getTestDB(t *testing.T) *sqlx.DB {
    // Starts postgres:17-alpine container
    // Runs migrations
    // Returns sqlx.DB connection
}
```

#### Table-Driven Tests

```go
func TestExample(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid input", "test", false},
        {"invalid input", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test logic
        })
    }
}
```

#### Helper Functions

Use `t.Helper()` for test helpers:

```go
func testRedisClient(t *testing.T) *redis.Client {
    t.Helper()
    // Setup logic
}
```

#### Skip When Dependencies Unavailable

```go
if testing.Short() {
    t.Skip("skipping integration test")
}
```

### Running Tests

```bash
# All tests
make test

# Verbose output
make test-verbose

# Race detector
make test-race

# Single test
make test-one PKG=./internal/middleware/ratelimit NAME=TestSlidingWindow

# Coverage
make coverage-html
```

---

## 8. API Documentation

### Swagger Location

API documentation is auto-generated via **Swag** and stored in:
- `backend/docs/docs.go`
- `backend/docs/swagger.json`
- `backend/docs/swagger.yaml`

### Viewing Documentation

When the API is running:
- **Swagger UI:** `http://localhost:8080/swagger/index.html`
- **JSON spec:** `http://localhost:8080/swagger/doc.json`

### Regenerating Documentation

```bash
make swag
```

This runs:
```bash
cd backend && swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
```

**Note:** Swag comments must be present on handlers for documentation to be complete.

---

## 9. Coding Conventions & Agent Guidelines

### Import Organization

Group imports in this order (blank line between groups):

```go
import (
    // 1. Standard library
    "context"
    "fmt"
    "log/slog"
    "net/http"
    "time"

    // 2. External packages
    "github.com/go-chi/chi/v5"
    "github.com/redis/go-redis/v9"
    "github.com/google/uuid"

    // 3. Internal packages
    "github.com/Yugsolanki/standfor-me/internal/config"
    "github.com/Yugsolanki/standfor-me/internal/domain"
    "github.com/Yugsolanki/standfor-me/internal/middleware"
)
```

### Formatting

- Use `gofumpt` (stricter `go fmt`)
- Run `make format` before committing
- golangci-lint auto-fixes formatting issues

### Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| Files | `snake_case` | `rate_limit.go`, `auth_handler_test.go` |
| Types/Functions | `PascalCase` | `User`, `NewUserService` |
| Variables/Methods | `camelCase` | `userRepo`, `getUserByID` |
| Constants | `PascalCase` or `SCREAMING_SNAKE_CASE` | `StatusActive`, `ErrNotFound` |
| Packages | Short, lowercase, no underscores | `repo`, `service`, `middleware` |

### Error Handling

**Always handle errors explicitly** — no `_` discard:

```go
// BAD
user, _ := h.service.GetUser(ctx, id)

// GOOD
user, err := h.service.GetUser(ctx, id)
if err != nil {
    return nil, fmt.Errorf("failed to get user: %w", err)
}
```

**Use domain error constructors:**

```go
if err == postgres.ErrNotFound {
    return nil, domain.NewNotFoundError("user_repository.get_by_id", "user not found")
}
```

**Wrap errors with context:**

```go
return nil, fmt.Errorf("failed to create limiter: %w", err)
```

### Structured Logging

Use `log/slog` with structured attributes:

```go
// GOOD
slog.Info("server started", "port", cfg.Server.Port)
slog.Error("connection failed", "error", err, "host", host)

// BAD
slog.Info(fmt.Sprintf("server started on port %d", cfg.Server.Port))
```

### HTTP Handlers

**Always use response helpers:**

```go
func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
    user, err := h.service.GetUser(r.Context(), id)
    if err != nil {
        response.JSONError(w, r, err)  // ONLY function for error responses
        return
    }
    response.JSON(w, r, http.StatusOK, user)
}
```

**Handler signature:**

```go
func (h *Handler) handlerName(w http.ResponseWriter, r *http.Request)
```

### Adding New Domain Entities

When adding a new domain entity (e.g., `Campaign`), follow this order:

1. **Domain types** (`internal/domain/campaign.go`)
   - Define struct, constants, param types
   - Add domain errors if needed

2. **Repository interfaces** (`internal/repository/postgres/campaign_repository.go`)
   - Implement CRUD operations
   - Add indexing method if searchable

3. **Service layer** (`internal/service/campaign_service.go`)
   - Business logic
   - Validation
   - Transaction coordination

4. **HTTP handlers** (`internal/server/campaign_handler.go`)
   - Request parsing
   - Response formatting
   - Error handling

5. **Routes** (`internal/server/routes.go`)
   - Add route definitions
   - Apply middleware

6. **Meilisearch integration** (if searchable)
   - Add document type (`internal/domain/search/document.go`)
   - Update `search.yaml`
   - Implement indexing in service

7. **Tests**
   - Unit tests for service
   - Integration tests for repository
   - Handler tests

### Middleware Order (When Adding Routes)

When adding new middleware or routes, maintain this order:

```go
r.Group(func(r chi.Router) {
    r.Use(middleware.RequireAuth(jwtSvc))            // Auth first
    r.Use(middleware.RequireMinRole(domain.RoleAdmin))  // Then role check
    // Add custom middleware here
    r.Get("/", handler)
})
```

### Validation

**Use struct tags for validation:**

```go
type CreateUserParams struct {
    Username string `validate:"required,min=3,max=30,username"`
    Email    string `validate:"required,email,max=255"`
    Password string `validate:"required,min=8,max=72"`
}
```

**Validate in handlers:**

```go
var req CreateMovementRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    response.JSONError(w, r, domain.NewBadRequestError("handler", "invalid JSON"))
    return
}

if errs := h.validator.Validate(&req); errs != nil {
    response.JSONError(w, r, domain.NewValidationError("handler", errs))
    return
}
```

### Redis Usage

**Use DB 15 for tests:**

```go
func testRedisClient(t *testing.T) *redis.Client {
    t.Helper()
    redisURL := os.Getenv("REDIS_URL")
    if redisURL == "" {
        redisURL = "localhost:6379"
    }
    // Use DB 15 for tests
    return redis.NewClient(&redis.Options{
        Addr:     redisURL,
        Password: os.Getenv("REDIS_PASSWORD"),
        DB:       15,
    })
}
```

**Rate limiter prefix:**

```
standfor:rl:global
standfor:rl:api:{endpoint}
```

### Git Hooks

**Pre-commit hook** (`.git/hooks/pre-commit`) runs on staged Go files:
1. `gofumpt` formatting check
2. `go vet` static analysis
3. Debug leftover scan (`fmt.Print`, `TODO:`, `FIXME:`, etc.)
4. Module consistency check

**Commit-msg hook** (`.git/hooks/commit-msg`) validates conventional commits:
```
type(scope): description
```

**Allowed types:** `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `ci`, `build`, `perf`

**Skip hooks:** `git commit --no-verify`

---

## 10. Roadmap Context

Understanding the three-phase MVP plan helps agents prioritize work and avoid scope creep.

### Phase 1 — Launch (Complete)

Core functionality for user profiles and movement browsing:

- ✅ User registration and authentication (JWT-based)
- ✅ User profiles with shareable links (`username.standfor.me`)
- ✅ Movement database with browse/search
- ✅ Add movements to personal list
- ✅ Public profile page (embeddable)
- ✅ Categories (Environment, Human Rights, etc.)
- ✅ Meilisearch integration for full-text search
- ✅ Basic admin panel (user management, movement moderation)

**Status:** Implemented and tested

### Phase 2 — Verification (In Progress)

Verification badge system implementation:

- 🔄 Verification badge data model
- 🔄 Badge tier logic (Bronze → Silver → Gold → Platinum)
- 🔄 Proof upload endpoints (donation receipts, event check-ins)
- 🔄 Organization vouching system
- 🔄 Social proof verification (hashtag tracking, org following)
- ⏳ AI verification engine (Python + Ollama) — future

**Status:** Data model in progress, verification logic pending

### Phase 3 — Discovery (Future)

Social and discovery features:

- ⏳ Find people with similar values
- ⏳ Movement pages with supporter counts
- ⏳ Trending movements algorithm
- ⏳ Organization claimed pages
- ⏳ Recommendation engine
- ⏳ Activity feed

**Status:** Not started

### Out of Scope (For Now)

- Frontend (Vue.js in separate repository)
- Mobile applications
- Payment processing for donations
- OAuth integrations (Twitter, Facebook, etc.)
- Real-time notifications (WebSocket)
- Advanced analytics dashboard

---

## Appendix: Quick Reference

### File Paths Cheat Sheet

| Purpose | Path |
|---------|------|
| Main entrypoint | `backend/cmd/api/main.go` |
| Routes | `backend/internal/server/routes.go` |
| Server setup | `backend/internal/server/server.go` |
| Domain types | `backend/internal/domain/*.go` |
| Domain errors | `backend/internal/domain/errors.go` |
| Auth middleware | `backend/internal/middleware/auth.go` |
| Rate limiter | `backend/internal/middleware/ratelimit/` |
| JWT service | `backend/internal/pkg/jwt/jwt.go` |
| Response helpers | `backend/internal/pkg/response/response.go` |
| User repository | `backend/internal/repository/postgres/user_repository.go` |
| User service | `backend/internal/service/user_service.go` |
| User handler | `backend/internal/server/user_handler.go` |
| Migrations | `backend/migrations/` |
| Config | `backend/configs/` |
| Swagger | `backend/docs/` |
| Tests | `*_test.go` alongside source |

### Common Commands

```bash
# Development
make run-dev              # Hot-reload server
make docker-up-infra      # Start postgres + redis
make migrate-up           # Run migrations

# Code quality
make format               # Format code
make lint                 # Run linter
make lint-fix             # Auto-fix issues

# Testing
make test                 # Run all tests
make test-race            # With race detector
make test-one PKG=... NAME=...  # Single test
make coverage-html        # Coverage report

# Build
make build                # Build API
make build-all            # Build all binaries
make swag                 # Regenerate Swagger docs

# Full pipeline
make all                  # tidy → format → lint → test → build
```

### Health Check Endpoints

- `GET /` — Root (returns JSON message)
- `GET /health` — Health check
- `GET /swagger/*` — Swagger UI

### API Base Path

All API routes are prefixed with `/api/v1`:
- `POST /api/v1/auth/register`
- `GET /api/v1/users/{username}`
- `GET /api/v1/movements`
- `GET /api/v1/search/movements`

---

*This document is maintained for AI coding agents. Update when architecture, conventions, or workflows change.*
