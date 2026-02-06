# Open Cloud Ops - Project Guide

## Overview

Open Cloud Ops (OCO) is an AI-powered FinOps platform with three modules:

| Module        | Language         | Port | Purpose                                             |
| ------------- | ---------------- | ---- | --------------------------------------------------- |
| **Cerebra**   | Go / Gin         | 8080 | LLM Gateway - intelligent routing, budgets, caching |
| **Economist** | Python / FastAPI | 8081 | Multi-cloud cost management & optimization          |
| **Aegis**     | Go / Gin         | 8082 | Kubernetes resilience - backup, recovery, DR        |

## Architecture

```
cerebra/          Go LLM reverse proxy (OpenAI, Anthropic, Gemini)
  cmd/main.go       Entry point
  internal/         Middleware (auth), proxy handler, router, analytics
  pkg/              Cache (Redis), database (Postgres), models

economist/        Python FinOps engine
  cmd/main.py       Entry point (FastAPI app factory)
  api/routes.py     REST endpoints (costs, recommendations, governance)
  internal/         Ingestion, optimizer, policy engines
  pkg/              Cloud providers (AWS/Azure/GCP), cost calculator, config

aegis/            Go K8s resilience
  cmd/main.go       Entry point
  api/handlers.go   REST endpoints + auth middleware
  internal/         Backup manager, storage backends, DR orchestrator

deploy/
  docker/           Docker Compose, Dockerfiles, init.sql
  kubernetes/       K8s manifests (StatefulSets, Deployments, Services)
```

## Tech Stack

- **Go 1.22+** with Gin web framework
- **Python 3.11+** with FastAPI, SQLAlchemy, Pydantic
- **PostgreSQL 16** + TimescaleDB extension (time-series hypertables)
- **Redis** for caching and rate limiting
- **Prometheus** + **Grafana** for observability

## Development

### Prerequisites

```bash
# Go modules
cd cerebra && go mod tidy
cd aegis && go mod tidy

# Python
cd economist && pip install -r requirements.txt
```

### Running Locally

```bash
# Full stack via Docker Compose (requires .env with passwords)
make docker-up

# Individual modules
make run-cerebra    # localhost:8080
make run-economist  # localhost:8081
make run-aegis      # localhost:8082
```

### Testing

```bash
make test           # All modules
make test-cerebra
make test-economist
make test-aegis
make lint           # All linters
```

### Environment Variables

Required (no defaults - will error if missing):

- `POSTGRES_PASSWORD` - Database password
- `GF_ADMIN_PASSWORD` - Grafana admin password

Optional with defaults:

- `POSTGRES_HOST` (localhost), `POSTGRES_PORT` (5432), `POSTGRES_DB` (opencloudops), `POSTGRES_USER` (oco_user)
- `REDIS_HOST` (localhost), `REDIS_PORT` (6379)
- `CORS_ALLOWED_ORIGINS` - Comma-separated list of allowed CORS origins

## API Authentication

All API endpoints require an `X-API-Key` header. Keys are stored hashed (SHA-256) in the `api_keys` table with a `key_prefix` index for lookup.

## Key Design Decisions

- **Constant-time hash comparison** via `crypto/subtle` to prevent timing attacks
- **Streaming backups** through temp files to avoid OOM on large archives
- **Buffered async logging** with channel + worker pattern for reliable request logging
- **Decimal arithmetic** for all currency calculations (no floating-point)
- **Non-root containers** (UID 1001) in all Dockerfiles
- **Required env vars** (`${VAR:?}`) for secrets - no default passwords

## Database

Schema defined in `deploy/docker/init.sql`. Key tables:

- `api_requests` - TimescaleDB hypertable for LLM request logs
- `api_keys` - Hashed API keys with prefix index
- `cloud_costs` - Cost data hypertable
- `optimization_recommendations`, `governance_policies`, `policy_violations`
- `backup_jobs`, `backup_records`, `dr_plans`
