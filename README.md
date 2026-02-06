# Open Cloud Ops

**An open-source AI FinOps platform for LLM cost tracking, cloud cost optimization, and cyber resilience.**

[![License: Apache-2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

> **Status: Phase 0 — Scaffolding & Design**
>
> This project is in its earliest stage. The module structure, data models, and API
> contracts are defined, but core functionality is not yet implemented. Contributions
> are welcome — see [Contributing](#contributing) and the [Roadmap](#roadmap) for where
> to start.

---

## Overview

Open Cloud Ops is a modular, open-source platform designed to give engineering, product, and finance teams **real-time visibility and control** over their AI and cloud infrastructure costs. It provides a unified solution for managing LLM API spend, optimizing cloud compute costs, and ensuring cyber resilience — all from a single dashboard.

The project aims to democratize access to enterprise-grade FinOps tooling through open-source collaboration.

## Architecture

Open Cloud Ops is built on three independent but integrated modules:

| Module | Code Name | Description | Status |
|--------|-----------|-------------|--------|
| **LLM Gateway** | Cerebra | A high-performance reverse proxy for LLM APIs. Tracks costs per agent, team, and model in real-time. Enforces hard budget limits and provides smart routing to reduce spend. | Scaffolded — data models and handler stubs defined |
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

## Planned Features

> The features below represent the project's design goals. They are **not yet
> implemented** unless otherwise noted. See the [Roadmap](#roadmap) for delivery
> phases.

### Cerebra — LLM Gateway

- **Real-Time Cost Tracking:** Monitor every dollar spent on LLM APIs, broken down by agent, team, model, and individual API call. *(data models defined)*
- **Hard Budget Controls:** Set monthly budgets per agent or team. Requests are automatically blocked when limits are reached. *(enforcer stub defined)*
- **Smart Model Routing:** Routes simple queries to cheaper models and complex queries to powerful models, maintaining quality while reducing costs. *(routing strategy types defined)*
- **AI-Powered Insights:** Proactive alerts for cost spikes, anomalies, and recommendations to switch to more cost-effective models. *(insight types defined)*
- **Multi-Provider Support:** Works with OpenAI, Anthropic (Claude), Google Gemini, and more. *(proxy handler stub supports these providers)*
- **Privacy-First:** API keys pass through in-memory only and are never stored. Only metadata (model, tokens, timestamps, costs) is logged.

### Economist — FinOps Core

- **Multi-Cloud Cost Visibility:** Unified dashboard for AWS, Azure, and GCP spend.
- **Optimization Recommendations:** Identifies idle resources, rightsizing opportunities, and spot/reserved instance savings.
- **Kubernetes Cost Monitoring:** Integrates with OpenCost for container-level cost allocation.
- **Governance & Policy:** Automated cost governance through Cloud Custodian integration.

### Aegis — Resilience Engine

- **Kubernetes Backup & Restore:** Powered by Velero for reliable cluster and volume backups.
- **Automated DR Policies:** Define backup schedules, retention rules, and recovery workflows.
- **Recovery Management:** Initiate and monitor recovery events from the dashboard.

## Getting Started

### Prerequisites

- Go 1.21+ (for Cerebra)
- Python 3.11+ (for Economist)
- Node.js 22+ with pnpm (for the Dashboard — not yet implemented)
- PostgreSQL 16+ with TimescaleDB
- Redis 7+
- Docker & Docker Compose (optional, for running dependencies)

### Running Services Locally

Since Docker Compose files and the dashboard are not yet available, individual
services can be run directly:

```bash
# Clone the repository
git clone https://github.com/bigdegenenergy/open-cloud-ops.git
cd open-cloud-ops
cp .env.example .env  # Configure your environment variables

# Run Cerebra (LLM Gateway) — prints startup banner, proxy not yet functional
cd cerebra && go run cmd/main.go

# Run Economist (FinOps Core) — starts FastAPI server with stub endpoints
cd economist && pip install -r requirements.txt && uvicorn cmd.main:app --reload --port 8081
```

> **Note:** Full `docker compose up` support is planned but the Compose files have
> not been created yet. Contributions to `deploy/docker/` are welcome.

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
│   ├── internal/             # Private application code
│   │   ├── proxy/            # Reverse proxy handler (stub)
│   │   ├── router/           # Smart model routing (stub)
│   │   ├── budget/           # Budget enforcement (stub)
│   │   ├── analytics/        # Cost analytics and insights (stub)
│   │   └── middleware/       # Auth, logging, rate limiting (empty)
│   ├── pkg/                  # Public packages
│   │   ├── models/           # Data models (defined)
│   │   ├── config/           # Configuration management (empty)
│   │   └── utils/            # Shared utilities (empty)
│   ├── api/                  # API definitions (empty)
│   └── web/dashboard/        # React frontend (empty)
├── economist/                # FinOps Core (Python)
│   ├── cmd/                  # FastAPI application (stub endpoints)
│   ├── internal/             # Private application code
│   │   ├── ingestion/        # Cloud cost data ingestion (empty)
│   │   ├── optimizer/        # Cost optimization engine (empty)
│   │   └── policy/           # Governance policy engine (empty)
│   ├── pkg/                  # Public packages
│   │   ├── cloud/            # Cloud provider integrations (empty)
│   │   └── cost/             # Cost calculation utilities (empty)
│   └── api/                  # API definitions (empty)
├── aegis/                    # Resilience Engine (empty)
│   ├── cmd/                  # Application entrypoint (empty)
│   ├── internal/             # Private application code (empty)
│   ├── pkg/                  # Public packages (empty)
│   └── api/                  # API definitions (empty)
├── docs/                     # Documentation (empty)
│   ├── architecture/         # Architecture decision records
│   ├── guides/               # User and developer guides
│   └── api-reference/        # API documentation
├── deploy/                   # Deployment configurations (empty)
│   ├── docker/               # Docker Compose files
│   └── kubernetes/           # Helm charts and K8s manifests
├── scripts/                  # Build, test, and utility scripts (empty)
└── .github/workflows/        # CI/CD pipelines (empty)
```

## Technology Stack

| Layer | Technology | Purpose |
|-------|-----------|---------|
| Proxy & Backend | Go (Gin) | High-performance LLM request proxying |
| API & Services | Python (FastAPI) | Flexible API development and data processing |
| Database | PostgreSQL + TimescaleDB | Time-series cost data and relational storage |
| Cache & Queue | Redis | Rate limiting, caching, async task management |
| Frontend | React + TailwindCSS + Chart.js | Interactive cost management dashboard |
| Observability | OpenTelemetry, Prometheus, Grafana | Monitoring, logging, and tracing |
| Cloud Cost | OpenCost, Cloud Custodian | Kubernetes and cloud cost monitoring |
| Disaster Recovery | Velero | Kubernetes backup and restore |

## Roadmap

- [x] **Phase 0 — Scaffolding:** Project structure, data models, API stubs, and documentation
- [ ] **Phase 1 — Cerebra MVP:** LLM proxy forwarding, cost tracking to database, budget enforcement, basic dashboard
- [ ] **Phase 2 — Economist:** Cloud cost ingestion (AWS/Azure/GCP), optimization recommendations, governance policies
- [ ] **Phase 3 — Aegis:** Kubernetes backup/restore via Velero, DR policy engine, recovery workflows
- [ ] **Phase 4 — Enterprise:** SSO, RBAC, advanced analytics, multi-tenancy

### What's included in Phase 0 (current)

- Module directory structure for Cerebra, Economist, and Aegis
- Go data models for API requests, agents, teams, organizations, and model pricing (`cerebra/pkg/models/`)
- Handler, router, budget enforcer, and analytics stubs with defined types and method signatures (`cerebra/internal/`)
- FastAPI application shell with health check and stub cost/recommendation/policy endpoints (`economist/cmd/main.py`)
- Dependency declarations (`go.mod`, `requirements.txt`)
- Environment variable template (`.env.example`)
- Contributing guidelines and Apache 2.0 license

## Contributing

We welcome contributions from the community! This project is early-stage, so there
are many opportunities to make a meaningful impact. High-value areas right now:

- Implementing core proxy logic in Cerebra (`cerebra/internal/proxy/handler.go`)
- Building out budget enforcement (`cerebra/internal/budget/enforcer.go`)
- Adding cloud cost ingestion to Economist (`economist/internal/ingestion/`)
- Creating Docker Compose files for local development (`deploy/docker/`)
- Setting up CI/CD with GitHub Actions (`.github/workflows/`)
- Writing tests for any module

Please read our [Contributing Guide](CONTRIBUTING.md) to get started.

## License

This project is licensed under the [Apache License 2.0](LICENSE).

## Acknowledgments

This project builds upon the incredible work of open-source projects including [LiteLLM](https://github.com/BerriAI/litellm), [OpenCost](https://opencost.io/), [Velero](https://velero.io/), and [Cloud Custodian](https://cloudcustodian.io/).
