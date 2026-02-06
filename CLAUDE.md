# CLAUDE.md — Open Cloud Ops

## Project Overview

Open Cloud Ops is an open-source AI FinOps platform with three modules:

| Module                        | Language                   | Port | Status      |
| ----------------------------- | -------------------------- | ---- | ----------- |
| **Cerebra** (LLM Gateway)     | Go (Gin)                   | 8080 | Phase 1 MVP |
| **Economist** (FinOps Core)   | Python (FastAPI)           | 8081 | Scaffolded  |
| **Aegis** (Resilience Engine) | Go                         | 8082 | Scaffolded  |
| **Dashboard**                 | React + Vite + TailwindCSS | 3000 | Phase 1 MVP |

## Build & Test Commands

```bash
# Cerebra (Go backend)
cd cerebra && go build ./...          # Build
cd cerebra && go test ./...           # Run tests
cd cerebra && gofmt -w .             # Format Go code

# Dashboard (React frontend)
cd dashboard && npm install           # Install deps
cd dashboard && npm run dev           # Dev server
cd dashboard && npm run build         # Production build

# Go module resolution (if proxy fails)
cd cerebra && GONOSUMCHECK=* GONOSUMDB=* GOPROXY=direct go mod tidy
```

## Architecture

### Cerebra — LLM Gateway

```
cerebra/
├── cmd/main.go                    # Entry point, wires all components
├── internal/
│   ├── api/handlers.go            # REST API endpoints
│   ├── analytics/insights.go      # Cost spike detection, model recommendations
│   ├── budget/enforcer.go         # Redis-backed budget enforcement
│   ├── config/config.go           # Environment variable loading
│   ├── database/
│   │   ├── database.go            # PostgreSQL connection, migrations, seed data
│   │   └── repository.go          # Query methods (insert, get, list)
│   ├── proxy/handler.go           # Reverse proxy for OpenAI/Anthropic/Gemini
│   └── router/router.go           # Smart model routing (cost/quality/latency)
└── pkg/models/models.go           # Shared data structures
```

### Key Design Decisions

- **API keys are never stored** — passed through in-memory only
- **Budget enforcement uses Redis** — sub-ms checks, graceful degradation when Redis unavailable
- **Cost tracking is async** — DB writes happen in goroutines to not block responses
- **Proxy is provider-agnostic** — same handler pattern for all 3 providers
- **Token extraction is provider-specific** — each provider has different response formats

### API Routes

```
GET  /health                              # Health check
ANY  /v1/proxy/openai/*path               # OpenAI proxy
ANY  /v1/proxy/anthropic/*path            # Anthropic proxy
ANY  /v1/proxy/gemini/*path               # Gemini proxy
GET  /api/v1/costs/summary                # Cost aggregation
GET  /api/v1/costs/requests               # Recent requests
GET  /api/v1/budgets                      # List budgets
POST /api/v1/budgets                      # Create budget
GET  /api/v1/budgets/:scope/:entity_id    # Get specific budget
GET  /api/v1/insights                     # Analytics insights
GET  /api/v1/report                       # Usage report
```

### Custom Headers

Clients identify themselves using headers on proxy requests:

- `X-Agent-ID` — Agent identifier (defaults to "default")
- `X-Team-ID` — Team identifier (defaults to "default")
- `X-Org-ID` — Organization identifier (defaults to "default")

Response includes cost metadata:

- `X-Request-ID` — Unique request identifier
- `X-Cost-USD` — Calculated cost for the request
- `X-Latency-Ms` — End-to-end latency

## Environment Variables

Key variables (see `.env.example` for full list):

| Variable                   | Default        | Description                                               |
| -------------------------- | -------------- | --------------------------------------------------------- |
| `CEREBRA_PORT`             | `8080`         | Server port                                               |
| `CEREBRA_ADMIN_API_KEY`    | _(empty)_      | API key for management endpoints (fail-secure when empty) |
| `CEREBRA_PROXY_API_KEY`    | _(empty)_      | Client key for proxy endpoints (open when empty)          |
| `CEREBRA_BUDGET_FAIL_OPEN` | `true`         | Allow requests when Redis unavailable                     |
| `POSTGRES_HOST`            | `localhost`    | PostgreSQL host                                           |
| `POSTGRES_PORT`            | `5432`         | PostgreSQL port                                           |
| `POSTGRES_DB`              | `opencloudops` | Database name                                             |
| `POSTGRES_USER`            | `oco_user`     | Database user                                             |
| `POSTGRES_PASSWORD`        | _(empty)_      | Database password                                         |
| `POSTGRES_SSLMODE`         | `prefer`       | PostgreSQL SSL mode                                       |
| `REDIS_HOST`               | `localhost`    | Redis host                                                |
| `REDIS_PORT`               | `6379`         | Redis port                                                |
| `OPENAI_API_KEY`           | _(empty)_      | Default OpenAI key                                        |
| `ANTHROPIC_API_KEY`        | _(empty)_      | Default Anthropic key                                     |
| `GOOGLE_API_KEY`           | _(empty)_      | Default Gemini key                                        |

## Development Notes

- Go 1.21+ required, but Go 1.24 is available in the dev environment
- Node.js 22 for the dashboard
- `go.mod`/`go.sum` contain version hashes that trigger false positive PII detection — this is expected
- Always run `gofmt -w` on Go files before committing
- Pre-commit hooks may block on false positives — use `--no-verify` when needed
- Docker Compose for local dev: `docker compose -f deploy/docker/docker-compose.dev.yml up -d`

## Phase Roadmap

- [x] Phase 0: Scaffolding & design
- [x] Phase 1: Cerebra MVP (proxy, cost tracking, budgets, dashboard)
- [ ] Phase 2: Economist (cloud cost ingestion, optimization, governance)
- [ ] Phase 3: Aegis (K8s backup/restore, DR policies)
- [ ] Phase 4: Enterprise (SSO, RBAC, multi-tenancy)
