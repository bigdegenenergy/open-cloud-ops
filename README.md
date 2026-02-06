# Open Cloud Ops

**An open-source AI FinOps platform for LLM cost tracking, cloud cost optimization, and cyber resilience.**

[![License: Apache-2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

---

## Overview

Open Cloud Ops is a modular, open-source platform designed to give engineering, product, and finance teams **real-time visibility and control** over their AI and cloud infrastructure costs. It provides a unified solution for managing LLM API spend, optimizing cloud compute costs, and ensuring cyber resilience — all from a single dashboard.

The project aims to democratize access to enterprise-grade FinOps tooling through open-source collaboration.

## Architecture

Open Cloud Ops is built on three independent but integrated modules:

| Module | Code Name | Description |
|--------|-----------|-------------|
| **LLM Gateway** | Cerebra | A high-performance reverse proxy for LLM APIs. Tracks costs per agent, team, and model in real-time. Enforces hard budget limits and provides AI-powered smart routing to reduce spend by 40-90%. |
| **FinOps Core** | Economist | A multi-cloud cost management engine. Integrates with AWS, Azure, and GCP to provide cost visibility, optimization recommendations, and governance frameworks. |
| **Resilience Engine** | Aegis | A Kubernetes-native backup and disaster recovery solution. Automates backup schedules, retention policies, and recovery workflows. |

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

## Key Features

### Cerebra — LLM Gateway

- **Real-Time Cost Tracking:** Monitor every dollar spent on LLM APIs, broken down by agent, team, model, and individual API call.
- **Hard Budget Controls:** Set monthly budgets per agent or team. Requests are automatically blocked when limits are reached — no surprises.
- **Smart Model Routing:** An AI-powered engine that routes simple queries to cheaper models (e.g., Claude Haiku, GPT-4o Mini) and complex queries to powerful models (e.g., Claude Opus, GPT-4), maintaining quality while reducing costs.
- **AI-Powered Insights:** Proactive alerts for cost spikes, anomalies, and recommendations to switch to more cost-effective models.
- **Multi-Provider Support:** Works with OpenAI, Anthropic (Claude), Google Gemini, and more.
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

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.21+
- Python 3.11+
- Node.js 22+
- PostgreSQL 16+ with TimescaleDB
- Redis 7+

### Run with Docker Compose

```bash
git clone https://github.com/bigdegenenergy/open-cloud-ops.git
cd open-cloud-ops
cp .env.example .env  # Configure your environment variables
docker compose up -d
```

The dashboard will be available at `http://localhost:3000`.

### Development Setup

```bash
# Clone the repository
git clone https://github.com/bigdegenenergy/open-cloud-ops.git
cd open-cloud-ops

# Start infrastructure services
docker compose -f deploy/docker/docker-compose.dev.yml up -d

# Run Cerebra (LLM Gateway)
cd cerebra && go run cmd/main.go

# Run Economist (FinOps Core)
cd economist && pip install -r requirements.txt && uvicorn cmd.main:app --reload

# Run the Dashboard
cd cerebra/web/dashboard && pnpm install && pnpm dev
```

## Project Structure

```
open-cloud-ops/
├── cerebra/                  # LLM Gateway module
│   ├── cmd/                  # Application entrypoint
│   ├── internal/             # Private application code
│   │   ├── proxy/            # Reverse proxy for LLM providers
│   │   ├── router/           # Smart model routing engine
│   │   ├── budget/           # Budget enforcement logic
│   │   ├── analytics/        # Cost analytics and insights
│   │   └── middleware/       # Auth, logging, rate limiting
│   ├── pkg/                  # Public packages
│   │   ├── models/           # Data models
│   │   ├── config/           # Configuration management
│   │   └── utils/            # Shared utilities
│   ├── api/                  # API definitions (OpenAPI/Swagger)
│   └── web/dashboard/        # React frontend
├── economist/                # FinOps Core module
│   ├── cmd/                  # Application entrypoint
│   ├── internal/             # Private application code
│   │   ├── ingestion/        # Cloud cost data ingestion
│   │   ├── optimizer/        # Cost optimization engine
│   │   └── policy/           # Governance policy engine
│   ├── pkg/                  # Public packages
│   │   ├── cloud/            # Cloud provider integrations
│   │   └── cost/             # Cost calculation utilities
│   └── api/                  # API definitions
├── aegis/                    # Resilience Engine module
│   ├── cmd/                  # Application entrypoint
│   ├── internal/             # Private application code
│   │   ├── backup/           # Backup orchestration
│   │   ├── recovery/         # Recovery workflows
│   │   └── policy/           # DR policy engine
│   ├── pkg/                  # Public packages
│   └── api/                  # API definitions
├── docs/                     # Documentation
│   ├── architecture/         # Architecture decision records
│   ├── guides/               # User and developer guides
│   └── api-reference/        # API documentation
├── deploy/                   # Deployment configurations
│   ├── docker/               # Docker Compose files
│   └── kubernetes/           # Helm charts and K8s manifests
├── scripts/                  # Build, test, and utility scripts
└── .github/workflows/        # CI/CD pipelines
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

- [x] **Phase 0:** Project scaffolding and documentation
- [ ] **Phase 1:** Cerebra MVP — LLM proxy, cost tracking, budget controls, dashboard
- [ ] **Phase 2:** Economist — Cloud cost ingestion, optimization recommendations
- [ ] **Phase 3:** Aegis — Kubernetes backup/restore, DR policy engine
- [ ] **Phase 4:** Enterprise features — SSO, RBAC, advanced analytics, multi-tenancy

## Contributing

We welcome contributions from the community! Please read our [Contributing Guide](CONTRIBUTING.md) to get started.

## License

This project is licensed under the [Apache License 2.0](LICENSE).

## Acknowledgments

This project builds upon the incredible work of open-source projects including [LiteLLM](https://github.com/BerriAI/litellm), [OpenCost](https://opencost.io/), [Velero](https://velero.io/), and [Cloud Custodian](https://cloudcustodian.io/).
