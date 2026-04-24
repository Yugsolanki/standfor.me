# ============================================================================
# Standfor.me - Makefile
# ============================================================================
# Go backend (chi + PostgreSQL + Redis), Vue.js frontend in a separate repo.
# Run `make help` to see all available targets.
# ============================================================================

# ---------- Variables --------------------------------------------------------

BINARY_NAME    := standfor-me
MIGRATE_NAME   := migrate
WORKER_NAME    := worker

BINARY_OUT     := backend/tmp/$(BINARY_NAME)
MIGRATE_OUT    := backend/tmp/$(MIGRATE_NAME)
WORKER_OUT     := backend/tmp/$(WORKER_NAME)

MODULE         := github.com/Yugsolanki/standfor-me

# Directories (relative to repo root)
BACKEND        := backend
CMD_API        := ./cmd/api
CMD_MIGRATE    := ./cmd/migrate
CMD_WORKER     := ./cmd/worker
MIGRATIONS_DIR := $(BACKEND)/migrations
SWAG_OUTPUT    := ./docs

# Docker / compose
COMPOSE_FILE   := docker-compose.yaml

# DB helpers
DOCKER_PG_USER := ${DATABASE_USER:-postgres}
DOCKER_PG_DB   := ${DATABASE_DBNAME:-standfor_dev}

# Test / coverage
COVER_PROFILE  := coverage.out
COVER_HTML     := coverage.html

# Tool binaries (installed via make install-tools when missing)
GOLANGCI_LINT  ?= $(shell command -v golangci-lint 2>/dev/null)
SWAG           ?= $(shell command -v swag 2>/dev/null)
GOFUMPT        ?= $(shell command -v gofumpt 2>/dev/null)

# Go flags
GO             := go
GO_BUILD_FLAGS := -v
GO_TAGS        :=
CGO_ENABLED    ?= 0

# Misc
DATE           := $(shell date +%Y-%m-%dT%H:%M:%S%z)
VERSION        ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT         := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

.PHONY: help \
	build build-api build-migrate build-worker build-all \
	run run-dev migrate-up migrate-down migrate-create \
	test test-verbose test-race test-coverage coverage-report \
	test-one \
	lint lint-fix \
	format tidy deps-update deps-cleanup \
	swag \
	docker-up docker-down docker-logs docker-clean \
	docker-build docker-build-all \
	goreleaser-snapshot goreleaser-release \
	install-tools \
	clean all

# ---------- Help -------------------------------------------------------------

help: ## Show this help summary
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN { FS = ":.*?## " }; { printf "\033[36m%-24s\033[0m %s\n", $$1, $$2 }'

# ---------- Build ------------------------------------------------------------

build: build-api ## Build the API binary (default target)
 
build-api: ## Build the API server binary
	@mkdir -p $(BACKEND)/tmp
	cd $(BACKEND) && CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GO_BUILD_FLAGS) -tags '$(GO_TAGS)' \
		-ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" \
		-o tmp/$(BINARY_NAME) $(CMD_API)
 
build-migrate: ## Build the migration CLI binary
	@mkdir -p $(BACKEND)/tmp
	cd $(BACKEND) && CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GO_BUILD_FLAGS) \
		-o tmp/$(MIGRATE_NAME) $(CMD_MIGRATE)
 
build-worker: ## Build the worker binary (asynq background worker)
	@mkdir -p $(BACKEND)/tmp
	cd $(BACKEND) && CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GO_BUILD_FLAGS) \
		-o tmp/$(WORKER_NAME) $(CMD_WORKER)

build-all: build-api build-migrate build-worker ## Build all binaries

# ---------- Run --------------------------------------------------------------

run: build-api ## Build and run the API server
	@./$(BINARY_OUT)

run-dev: ## Hot-reload with air (requires `go install github.com/air-verse/air@latest`)
	cd $(BACKEND) && air

# ---------- Database Migrations ----------------------------------------------

migrate-up: ## Run all pending migrations (via `go run ./cmd/migrate`)
	cd $(BACKEND) && $(GO) run $(CMD_MIGRATE)

migrate-down: ## Roll back the last applied migration
	cd $(BACKEND) && $(GO) run $(CMD_MIGRATE) -direction down

migrate-status: ## Show current migration status
	@echo "Migration files in $(MIGRATIONS_DIR):"
	@ls -1 $(MIGRATIONS_DIR)/

migrate-create: ## Create a new migration file (usage: make migrate-create NAME=create_foo)
	@if [ -z "$(NAME)" ]; then \
		echo "Usage: make migrate-create NAME=create_foo"; exit 1; fi
	@latest=$$(ls -1 $(MIGRATIONS_DIR)/*.up.sql 2>/dev/null | \
		xargs -n1 basename 2>/dev/null | \
		grep -oE '^[0-9]+' | sort -n | tail -1); \
	next=$$(printf "%06d" $$((10#$${latest:-0} + 1))); \
	touch $(MIGRATIONS_DIR)/$${next}_$(NAME).up.sql \
	      $(MIGRATIONS_DIR)/$${next}_$(NAME).down.sql && \
	echo "✓ Created migration: $${next}_$(NAME)"

# ---------- Tests ------------------------------------------------------------

test: ## Run all tests
	cd $(BACKEND) && $(GO) test ./...

test-verbose: ## Run all tests with verbose output
	cd $(BACKEND) && $(GO) test -v ./...

test-race: ## Run all tests with the race detector
	cd $(BACKEND) && $(GO) test -race ./...

test-coverage: ## Run tests and write a coverage profile
	cd $(BACKEND) && $(GO) test -coverprofile=$(COVER_PROFILE) -covermode=atomic ./...

coverage-report: test-coverage ## Generate HTML coverage report
	cd $(BACKEND) && $(GO) tool cover -html=$(COVER_PROFILE) -o $(COVER_HTML)

coverage-text: test-coverage ## Print coverage summary to terminal
	cd $(BACKEND) && $(GO) tool cover -func=$(COVER_PROFILE)

coverage-html: coverage-report ## Generate and open HTML coverage report
	@if command -v xdg-open >/dev/null 2>&1; then \
		xdg-open $(BACKEND)/$(COVER_HTML); \
	elif command -v open >/dev/null 2>&1; then \
		open $(BACKEND)/$(COVER_HTML); \
	else \
		echo "Open $(BACKEND)/$(COVER_HTML) manually."; \
	fi

test-one: ## Run a single test (usage: make test-one PKG=./internal/middleware/ratelimit NAME=TestSlidingWindow)
	cd $(BACKEND) && $(GO) test -v $(PKG) -run $(NAME)

# ---------- Linting & Formatting ---------------------------------------------

lint: ## Run golangci-lint
	cd $(BACKEND) && golangci-lint run

lint-fix: ## Run golangci-lint with auto-fix
	cd $(BACKEND) && golangci-lint run --fix

format: ## Run gofumpt across the backend
	@if [ -z "$(GOFUMPT)" ]; then \
		echo "Installing gofumpt..."; \
		$(GO) install mvdan.cc/gofumpt@latest; \
	fi
	cd $(BACKEND) && gofumpt -l -w .

# ---------- Modules & Dependencies -------------------------------------------

tidy: ## Run `go mod tidy`
	cd $(BACKEND) && $(GO) mod tidy

deps-update: ## Update all direct dependencies to latest minor/patch
	cd $(BACKEND) && $(GO) get -u ./... && $(GO) mod tidy

deps-cleanup: ## Remove unused modules from go.sum
	cd $(BACKEND) && $(GO) mod tidy

deps-vuln: ## Check dependencies for known vulnerabilities (govulncheck)
	@if command -v govulncheck >/dev/null 2>&1; then \
		cd $(BACKEND) && govulncheck ./...; \
	else \
		echo "Installing golang.org/x/vuln/cmd/govulncheck..."; \
		$(GO) install golang.org/x/vuln/cmd/govulncheck@latest; \
		cd $(BACKEND) && govulncheck ./...; \
	fi

# ---------- Swagger / API Docs -----------------------------------------------

swag: ## Generate Swagger documentation (swag init)
	@if [ -z "$(SWAG)" ]; then \
		echo "Installing swag..."; \
		$(GO) install github.com/swaggo/swag/cmd/swag@latest; \
	fi
	cd $(BACKEND) && swag init -g $(CMD_API)/main.go -o $(SWAG_OUTPUT) --parseDependency --parseInternal

# ---------- Docker & Compose -------------------------------------------------

docker-up: ## Start all services (postgres, redis, pgadmin, redisinsight)
	docker compose -f $(COMPOSE_FILE) up -d

docker-up-infra: ## Start only infrastructure (postgres + redis, skip UI tools)
	docker compose -f $(COMPOSE_FILE) up -d postgres redis

docker-down: ## Stop and remove all containers
	docker compose -f $(COMPOSE_FILE) down

docker-logs: ## Tail logs from all containers
	docker compose -f $(COMPOSE_FILE) logs -f

docker-logs-svc: ## Tail logs from a specific service (usage: make docker-logs-svc SVC=postgres)
	docker compose -f $(COMPOSE_FILE) logs -f $(SVC)

docker-clean: ## Stop containers and remove volumes
	docker compose -f $(COMPOSE_FILE) down -v

docker-build: ## Build Docker images using GoReleaser snapshots
	docker compose -f $(COMPOSE_FILE) build

# ---------- GoReleaser -------------------------------------------------------

goreleaser-snapshot: ## Build local snapshot release (binaries + Docker images)
	cd $(BACKEND) && goreleaser release --snapshot --clean

goreleaser-release: ## Build and publish a release (requires valid GitHub token)
	cd $(BACKEND) && goreleaser release --clean

# ---------- Tool Installation -------------------------------------------------

install-tools: ## Install common dev tools (golangci-lint, swag, gofumpt, air)
	$(GO) install github.com/swaggo/swag/cmd/swag@latest
	$(GO) install mvdan.cc/gofumpt@latest
	$(GO) install github.com/air-verse/air@latest
	$(GO) install github.com/goreleaser/goreleaser/v2@latest
	$(GO) install golang.org/x/vuln/cmd/govulncheck@latest

# ---------- Clean ------------------------------------------------------------

clean: ## Remove build artifacts and coverage files
	@rm -rf $(BACKEND)/tmp
	@rm -f $(BACKEND)/$(COVER_PROFILE) $(BACKEND)/$(COVER_HTML)
	@echo "Cleaned build artifacts and coverage files."

# ---------- Convenience Aliases ----------------------------------------------

all: tidy format lint test build ## Full pipeline: tidy -> format -> lint -> test -> build
