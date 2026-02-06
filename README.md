# Open Cloud Ops

**An open-source AI FinOps platform for LLM cost tracking, cloud cost optimization, and cyber resilience.**

[![License: Apache-2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

> **Status: Phase 1 — Cerebra MVP**
>
> The Cerebra LLM Gateway is now functional with proxy forwarding, cost tracking,
> budget enforcement, and a basic React dashboard. See the [Roadmap](#roadmap) for
> what's next.

---

## Overview

Open Cloud Ops is a modular, open-source platform designed to give engineering, product, and finance teams **real-time visibility and control** over their AI and cloud infrastructure costs. It provides a unified solution for managing LLM API spend, optimizing cloud compute costs, and ensuring cyber resilience — all from a single dashboard.

The project aims to democratize access to enterprise-grade FinOps tooling through open-source collaboration.

## Architecture

Open Cloud Ops is built on three independent but integrated modules:

| Module | Code Name | Description | Status |
|--------|-----------|-------------|--------|
| **LLM Gateway** | Cerebra | A high-performance reverse proxy for LLM APIs. Tracks costs per agent, team, and model in real-time. Enforces hard budget limits and provides smart routing to reduce spend. | **MVP implemented** — proxy, cost tracking, budget enforcement, API, dashboard |
| **FinOps Core** | Economist | A multi-cloud cost management engine. Integrates with AWS, Azure, and GCP to provide cost visibility, optimization recommendations, and governance frameworks. | Scaffolded — FastAPI shell with endpoint stubs |
| **Resilience Engine** | Aegis | A Kubernetes-native backup and disaster recovery solution. Automates backup schedules, retention policies, and recovery workflows. | Directory structure only |

```
┌─────────────────────────────────────────────────────────┐
│                    React Dashboard                       │
├─────────────────────────────────────────────────────────┤
│                    API Gateway                           │
├──────────────┬──────────────────┬───────────────────────┤
│   Cerebra    │    Economist     │        Aegis          │
│  LLM Gateway │   FinOps Core   │  Resilience Engine    │
├──────────────┴──────────────────┴───────────────────────┤
│  PostgreSQL + TimescaleDB │ Redis │ Prometheus + Grafana │
└─────────────────────────────────────────────────────────┘
```

## Features

### Cerebra — LLM Gateway (Phase 1 MVP)

- **LLM Proxy Forwarding:** Reverse proxy for OpenAI, Anthropic, and Google Gemini APIs. Routes requests transparently to upstream providers and returns responses unchanged.
- **Real-Time Cost Tracking:** Every proxied request is logged with model, token counts, cost, latency, and agent/team metadata. Costs are calculated using per-model pricing tables.
- **Hard Budget Controls:** Set monthly budgets per agent, team, or organization. Requests are automatically blocked (HTTP 402) when limits are reached. Enforced via Redis for sub-millisecond checks.
- **Smart Model Routing:** Routes simple queries to cheaper models and complex queries to powerful models, supporting cost-optimized, quality-first, and latency-optimized strategies.
- **AI-Powered Insights:** Detects cost spikes (2x rolling average), recommends cheaper model alternatives, and generates usage reports.
- **Multi-Provider Support:** Works with OpenAI, Anthropic (Claude), and Google Gemini.
- **Privacy-First:** API keys pass through in-memory only and are never stored. Only metadata (model, tokens, timestamps, costs) is logged.
- **REST API:** Full API for cost summaries, budget management, insights, and usage reports.
- **React Dashboard:** Interactive frontend for cost visualization, budget management, and insights.

### Economist — FinOps Core (planned)

- **Multi-Cloud Cost Visibility:** Unified dashboard for AWS, Azure, and GCP spend.
- **Optimization Recommendations:** Identifies idle resources, rightsizing opportunities, and spot/reserved instance savings.
- **Kubernetes Cost Monitoring:** Integrates with OpenCost for container-level cost allocation.
- **Governance & Policy:** Automated cost governance through Cloud Custodian integration.

### Aegis — Resilience Engine (planned)

- **Kubernetes Backup & Restore:** Powered by Velero for reliable cluster and volume backups.
- **Automated DR Policies:** Define backup schedules, retention rules, and recovery workflows.
- **Recovery Management:** Initiate and monitor recovery events from the dashboard.

## Getting Started

### Prerequisites

- Go 1.21+ (for Cerebra)
- Python 3.11+ (for Economist)
- Node.js 22+ (for the Dashboard)
- PostgreSQL 16+
- Redis 7+
- Docker & Docker Compose (recommended for local development)

### Quick Start with Docker Compose

```bash
# Clone the repository
git clone https://github.com/bigdegenenergy/open-cloud-ops.git
cd open-cloud-ops
cp .env.example .env  # Configure your environment variables

# Start all services
docker compose -f deploy/docker/docker-compose.dev.yml up -d

# Services will be available at:
# Cerebra API:  http://localhost:8080
# Dashboard:    http://localhost:3000
# Health check: http://localhost:8080/health
```

### Running Services Directly

```bash
# Clone and configure
git clone https://github.com/bigdegenenergy/open-cloud-ops.git
cd open-cloud-ops
cp .env.example .env

# Run Cerebra (LLM Gateway)
cd cerebra && go run cmd/main.go

# Run Dashboard
cd dashboard && npm install && npm run dev

# Run Economist (FinOps Core) — stub endpoints only
cd economist && pip install -r requirements.txt && uvicorn cmd.main:app --reload --port 8081
```

### API Usage

#### Proxy an LLM request through Cerebra

```bash
# OpenAI
curl -X POST http://localhost:8080/v1/proxy/openai/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-openai-key" \
  -H "X-Agent-ID: my-agent" \
  -H "X-Team-ID: my-team" \
  -d '{"model": "gpt-4o", "messages": [{"role": "user", "content": "Hello"}]}'

# Anthropic
curl -X POST http://localhost:8080/v1/proxy/anthropic/v1/messages \
  -H "Content-Type: application/json" \
  -H "X-API-Key: sk-ant-your-key" \
  -H "X-Agent-ID: my-agent" \
  -d '{"model": "claude-sonnet-4-20250514", "max_tokens": 1024, "messages": [{"role": "user", "content": "Hello"}]}'
```

#### Query cost data

```bash
# Cost summary by model
curl http://localhost:8080/api/v1/costs/summary?dimension=model

# Recent requests
curl http://localhost:8080/api/v1/costs/requests?limit=20

# Insights and recommendations
curl http://localhost:8080/api/v1/insights

# Usage report
curl http://localhost:8080/api/v1/report
```

#### Manage budgets

```bash
# Create a budget
curl -X POST http://localhost:8080/api/v1/budgets \
  -H "Content-Type: application/json" \
  -d '{"scope": "agent", "entity_id": "my-agent", "limit_usd": 100.0, "period_days": 30}'

# List budgets
curl http://localhost:8080/api/v1/budgets

# Get specific budget
curl http://localhost:8080/api/v1/budgets/agent/my-agent
```

### Environment Variables

Copy `.env.example` to `.env` and update the values. Key settings:

| Variable | Default | Description |
|----------|---------|-------------|
| `CEREBRA_PORT` | `8080` | Port for the LLM Gateway |
| `ECONOMIST_PORT` | `8081` | Port for the FinOps Core API |
| `AEGIS_PORT` | `8082` | Port for the Resilience Engine |
| `DASHBOARD_PORT` | `3000` | Port for the React dashboard |
| `POSTGRES_HOST` | `localhost` | PostgreSQL host |
| `POSTGRES_PORT` | `5432` | PostgreSQL port |
| `REDIS_HOST` | `localhost` | Redis host |

See `.env.example` for the full list, including optional cloud provider credentials
and observability settings.

## Project Structure

```
open-cloud-ops/
├── cerebra/                  # LLM Gateway (Go)
│   ├── cmd/                  # Application entrypoint
│   ├── internal/
│   │   ├── api/              # REST API handlers
│   │   ├── analytics/        # Cost spike detection and insights
│   │   ├── budget/           # Redis-backed budget enforcement
│   │   ├── config/           # Configuration from environment
│   │   ├── database/         # PostgreSQL connection, migrations, repository
│   │   ├── proxy/            # Reverse proxy for LLM providers
│   │   └── router/           # Smart model routing engine
│   └── pkg/models/           # Shared data models
├── dashboard/                # React + Vite + TailwindCSS frontend
│   └── src/
│       ├── components/       # Dashboard UI components
│       └── api.ts            # API client
├── economist/                # FinOps Core (Python/FastAPI)
│   ├── cmd/                  # FastAPI application (stub endpoints)
│   └── internal/             # Private application code
├── aegis/                    # Resilience Engine (planned)
├── deploy/
│   └── docker/               # Docker Compose and Dockerfiles
├── docs/                     # Documentation
└── .github/workflows/        # CI/CD pipelines
```

## Technology Stack

| Layer | Technology | Purpose |
|-------|-----------|---------|
| Proxy & Backend | Go (Gin) | High-performance LLM request proxying |
| API & Services | Python (FastAPI) | Flexible API development and data processing |
| Database | PostgreSQL | Relational storage for cost data and budgets |
| Cache & Budget | Redis | Budget enforcement and spend tracking |
| Frontend | React + TailwindCSS + Chart.js | Interactive cost management dashboard |
| Observability | OpenTelemetry, Prometheus, Grafana | Monitoring, logging, and tracing |
| Cloud Cost | OpenCost, Cloud Custodian | Kubernetes and cloud cost monitoring |
| Disaster Recovery | Velero | Kubernetes backup and restore |

## Roadmap

- [x] **Phase 0 — Scaffolding:** Project structure, data models, API stubs, and documentation
- [x] **Phase 1 — Cerebra MVP:** LLM proxy forwarding, cost tracking to database, budget enforcement, basic dashboard
- [ ] **Phase 2 — Economist:** Cloud cost ingestion (AWS/Azure/GCP), optimization recommendations, governance policies
- [ ] **Phase 3 — Aegis:** Kubernetes backup/restore via Velero, DR policy engine, recovery workflows
- [ ] **Phase 4 — Enterprise:** SSO, RBAC, advanced analytics, multi-tenancy

### What's included in Phase 1

- **Proxy handler** — Full reverse proxy for OpenAI, Anthropic, and Gemini APIs with header forwarding, auth passthrough, and response streaming
- **Cost tracking** — Automatic token extraction from provider responses, cost calculation using model pricing tables, async database writes
- **Budget enforcement** — Redis-backed budget checks with sub-ms latency, configurable per agent/team/org, automatic request blocking at limits
- **Database layer** — PostgreSQL schema with migrations, model pricing seed data, repository pattern for all queries
- **REST API** — Endpoints for cost summaries, recent requests, budget CRUD, insights, and usage reports
- **Analytics engine** — Cost spike detection using rolling averages, model switch recommendations with estimated savings
- **Smart routing** — Cost-optimized, quality-first, and latency-optimized routing strategies with model tier classification
- **React dashboard** — Cost overview cards, breakdown charts, request history, budget management, and insights panel
- **Docker Compose** — Development environment with PostgreSQL, Redis, Cerebra, and dashboard services
- **Unit tests** — Tests for proxy handler, router, config, and budget enforcer

## Contributing

We welcome contributions from the community! High-value areas right now:

- Adding cloud cost ingestion to Economist (`economist/internal/ingestion/`)
- Implementing streaming proxy support for SSE responses
- Adding more comprehensive integration tests
- Setting up CI/CD with GitHub Actions (`.github/workflows/`)
- Improving the dashboard with more visualizations
- Adding Prometheus metrics export

Please read our [Contributing Guide](CONTRIBUTING.md) to get started.

## License

This project is licensed under the [Apache License 2.0](LICENSE).

## Acknowledgments

This project builds upon the incredible work of open-source projects including [LiteLLM](https://github.com/BerriAI/litellm), [OpenCost](https://opencost.io/), [Velero](https://velero.io/), and [Cloud Custodian](https://cloudcustodian.io/).
