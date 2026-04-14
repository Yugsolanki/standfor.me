# Standfor.me

## Project Overview

Standfor.me is a digital platform designed to transform passive sympathy into visible, verified advocacy. It serves as a centralized public profile where individuals can declare the social movements and causes they stand behind, moving beyond the noise of traditional social media to create a clear, permanent record of personal values.

## The Problem:

In the current digital landscape, support for social causes is often fragmented, performative, or hidden within algorithmic feeds. Individuals care deeply about issues ranging from climate justice to human rights, but there is no standardized way to aggregate that support or prove its authenticity. This creates a "credibility gap" where it is difficult to distinguish between genuine activists and those engaging in surface-level alignment, making it hard for organizations to find true allies and for individuals to find like-minded peers.

## The Solution

Standfor.me provides users with a dedicated space to curate a "advocacy portfolio." Through a unique, shareable profile link, users can build a public list of the movements they support. Unlike a simple social media bio, this platform introduces a structured verification system that quantifies the depth of a user's commitment, allowing them to demonstrate how they support a cause, not just that they support it.

The Verification Hierarchy

The core innovation of the project is a tiered badge system that validates different levels of engagement, creating a trust layer for digital activism:

* Self-Declared (Entry Level): Users can publicly list causes they align with, creating a digital declaration of their values.
* Social Proof (Bronze): This tier validates support through existing digital footprints, such as following relevant organizations or using specific advocacy hashtags, proving that the user is part of the digital conversation.
* Financial Proof (Silver): Users can verify tangible support through charitable contributions, connecting platforms like Patreon or uploading donation receipts to prove they have put resources behind their beliefs.
* Action Proof (Gold): This tier recognizes physical real-world engagement, such as attending rallies, verified check-ins at events, or logging volunteer hours, bridging the gap between online presence and offline action.
* Organization Vouched (Platinum): The highest level of verification, where verified non-profits or NGOs officially vouch for a user’s role as a volunteer, ambassador, or staff member.

## Why Matters

Standfor.me aims to solve the "slacktivism" problem by providing a mechanism for accountability. It allows users to build a reputation based on their values, offering a "resume for activism" that can be shared with communities, potential employers, or friends. By quantifying support, the platform fosters a culture of transparency, helping users discover peers with shared values and allowing movements to identify their most dedicated supporters.

Core MVP Features:

A platform for users to publicly share and verify their support for social movements/causes with shareable profiles.

```md
Phase 1 - Launch
├── User profiles with shareable links (username.standfor.me)
├── Browse/search movement database
├── Add movements to personal list
├── Public profile page (embeddable)
├── Basic categories (Environment, Human Rights, etc.)
└── Social sharing buttons

Phase 2 - Verification
├── Verification badges system
├── Connect external accounts (MOSTLY NOT)
├── Proof uploads
└── Community vouching (NOT SURE)

Phase 3 - Discovery
├── Find people with similar values
├── Movement pages with supporter counts
├── Trending movements
└── Organization claimed pages
```

## Tech Stack

Backend: Golang
Frontend: Vue.js
AI Verification Engine: Python + Ollama
Database: Postgres

## Development

### Prerequisites

- **Go** 1.25.5+
- **Docker** & **Docker Compose**
- **Make**

### Quick Start

```bash
# 1. Install dev tools (golangci-lint, swag, gofumpt, air, goreleaser, govulncheck)
make install-tools

# 2. Start Postgres and Redis
make docker-up-infra

# 3. Run database migrations
make migrate-up

# 4. Start the API with hot-reload
make run-dev
```

### Common Commands

All operations go through the **Makefile** at the repo root. Run `make help` for the full list.

| Command | Description |
|---|---|
| `make run-dev` | Hot-reload dev server (requires air) |
| `make lint` | Run golangci-lint |
| `make format` | Format all Go files (gofumpt) |
| `make test` | Run all tests |
| `make test-race` | Run tests with race detector |
| `make coverage-html` | Generate and open HTML coverage report |
| `make swag` | Generate Swagger API docs |
| `make migrate-up` | Apply pending migrations |
| `make migrate-create NAME=x` | Create a new migration pair |
| `make docker-down` | Stop all containers |
| `make all` | Full pipeline: tidy → format → lint → test → build |

### Project Structure

```
.
├── Makefile                  # Build, test, lint, docker targets
├── docker-compose.yaml       # Postgres, Redis, pgAdmin, RedisInsight
├── backend/
│   ├── cmd/api/              # Main API entrypoint
│   ├── cmd/migrate/          # Migration CLI
│   ├── cmd/worker/           # Asynq background worker
│   ├── internal/server/       # Chi router & HTTP server
│   ├── internal/service/      # Business logic
│   ├── internal/repository/   # Postgres & Redis data access
│   ├── internal/middleware/   # HTTP middleware stack
│   ├── migrations/            # SQL migration files
│   └── docs/                  # Swagger-generated docs
└── .env                       # Environment variables
```

See [AGENTS.md](./AGENTS.md) for comprehensive development guidelines, code style, and patterns.
