# ADR-001: Modular Three-Service Architecture

**Status:** Accepted
**Date:** 2026-02-06
**Deciders:** Open Cloud Ops Core Team
**Category:** Architecture

---

## Context

Open Cloud Ops addresses three distinct operational domains:

1. **LLM cost management** -- tracking, budgeting, and optimizing spend on LLM API providers (OpenAI, Anthropic, Google Gemini)
2. **Multi-cloud FinOps** -- ingesting, analyzing, and optimizing infrastructure costs across AWS, Azure, and GCP
3. **Kubernetes resilience** -- automating backup, disaster recovery, and health monitoring for Kubernetes clusters

These domains have fundamentally different characteristics:

- **Performance profiles:** LLM proxying requires sub-millisecond overhead on a high-throughput hot path. Cloud cost ingestion is a batch workload that runs periodically. Backup operations are scheduled and latency-tolerant.
- **Language ecosystems:** LLM proxying benefits from Go's raw HTTP performance. Cloud cost analysis benefits from Python's rich SDK ecosystem (boto3, azure-sdk, google-cloud libraries) and data analysis tools. Kubernetes tooling is predominantly written in Go.
- **Scaling patterns:** The LLM gateway needs horizontal scaling to handle concurrent proxy requests. The FinOps engine needs worker scaling for batch ingestion. The resilience engine scales with the number of managed clusters.
- **Release cadences:** Each domain evolves independently. LLM provider APIs change frequently. Cloud billing APIs have their own release cycles. Kubernetes backup tooling follows the Kubernetes release cadence.

A monolithic architecture would force all three domains into a single language, a single deployment unit, and a single scaling strategy, creating unnecessary coupling and operational complexity.

## Decision

We adopt a **modular three-service architecture** with the following independently deployable modules:

| Module | Code Name | Language | Port | Domain |
|--------|-----------|----------|------|--------|
| LLM Gateway | **Cerebra** | Go (Gin) | 8080 | LLM proxy, cost tracking, budget enforcement, smart routing, analytics |
| FinOps Core | **Economist** | Python (FastAPI) | 8081 | Multi-cloud cost ingestion, optimization recommendations, governance policies |
| Resilience Engine | **Aegis** | Go (Gin) | 8082 | Kubernetes backup, recovery workflows, DR policies, health monitoring |

### Shared Infrastructure

All three modules share a common data layer to enable cross-module analytics and a unified dashboard:

- **PostgreSQL + TimescaleDB** -- Primary data store for all modules. Cerebra uses TimescaleDB hypertables for time-series request data. Economist uses standard relational tables with JSONB for flexible policy rules.
- **Redis** -- Shared caching layer for budget lookups (Cerebra), rate limiting, and task queue brokering (Economist/Celery).
- **Prometheus + Grafana** -- Unified observability stack collecting metrics from all modules via OpenTelemetry.

### Communication Pattern

Modules communicate indirectly through the shared database rather than through direct service-to-service RPC. This keeps modules decoupled and eliminates cascading failure scenarios. If Economist is down, Cerebra continues proxying LLM requests without interruption.

```
  +----------+     +----------+     +----------+
  |  Cerebra |     | Economist|     |   Aegis  |
  +----+-----+     +----+-----+     +----+-----+
       |                |                |
       v                v                v
  +----+----------------+----------------+-----+
  |              PostgreSQL + Redis             |
  +--------------------------------------------+
```

### Project Structure Convention

Each module follows a consistent internal structure regardless of language:

```
<module>/
  cmd/           -- Application entrypoint (main.go or main.py)
  internal/      -- Private implementation packages
  pkg/           -- Public/shared packages (models, config, database)
  api/           -- API definitions (OpenAPI specs)
```

This consistency reduces cognitive overhead for contributors working across modules.

## Consequences

### Positive

- **Independent scaling:** Each module scales based on its own load profile. Cerebra can scale to 20 replicas to handle proxy traffic while Aegis runs 2 replicas for scheduled backups.
- **Language-specific optimization:** Go gives Cerebra the raw HTTP performance needed for low-latency proxying. Python gives Economist access to boto3, azure-sdk, and data analysis libraries without FFI overhead.
- **Independent deployability:** A bug fix in Economist can be deployed without restarting Cerebra. This reduces deployment risk and enables faster iteration.
- **Fault isolation:** If Economist crashes during a cost ingestion run, Cerebra continues proxying LLM requests and Aegis continues managing backups. No single module failure takes down the entire platform.
- **Team autonomy:** Different teams or contributors can own different modules with domain-specific expertise (Go/infrastructure vs. Python/data engineering).
- **Flexible adoption:** Organizations can deploy only the modules they need. A team that only needs LLM cost tracking can deploy Cerebra alone without Economist or Aegis.

### Negative

- **Operational complexity:** Three services means three deployment pipelines, three sets of health checks, and three log streams to monitor. Mitigated by Docker Compose for development and Kubernetes with Helm charts for production.
- **Shared database coupling:** While modules are independently deployable, they share a PostgreSQL instance. Schema migrations must be coordinated to avoid breaking other modules. Mitigated by each module owning its own set of tables with no cross-module foreign keys.
- **Data consistency:** Without direct service-to-service calls, cross-module data consistency relies on eventual consistency through the shared database. This is acceptable for the analytics use case but would need revision if real-time cross-module coordination is required.
- **Duplicated infrastructure code:** Some patterns (config loading, database connection pooling, health checks) are implemented separately in Go and Python. Mitigated by keeping shared patterns minimal and well-documented.

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Schema migration conflicts | Each module owns its tables. No cross-module foreign keys. Migrations are versioned per module. |
| Inconsistent API patterns | Shared API design guidelines. All modules use `/health` for health checks and `/api/v1/` prefix for business endpoints. |
| Configuration drift | Shared `.env.example` with all variables. Docker Compose ensures consistent infrastructure configuration. |
| Monitoring blind spots | Unified Prometheus/Grafana stack with per-module dashboards and a cross-module overview dashboard. |

## Alternatives Considered

### 1. Monolithic Application (Go)

Build all three domains into a single Go binary.

- **Rejected because:** Python's cloud SDK ecosystem is significantly richer than Go's for multi-cloud cost management. Forcing all domains into Go would require substantial wrapper code around cloud provider APIs. A monolith would also couple the release and scaling of all three domains.

### 2. Monolithic Application (Python)

Build all three domains into a single Python application.

- **Rejected because:** Python's HTTP proxying performance is insufficient for the LLM gateway use case, where every millisecond of added latency directly impacts user experience. The proxy must handle thousands of concurrent requests with minimal overhead.

### 3. Microservices with RPC (gRPC)

Decompose into many fine-grained microservices communicating via gRPC.

- **Rejected because:** The current scope does not warrant the operational complexity of a full microservices architecture with service mesh, circuit breakers, and distributed tracing across dozens of services. Three modules with shared storage strikes the right balance for the current team size and feature set. This decision can be revisited as the project grows.

### 4. Two Services (Go + Python)

Combine Cerebra and Aegis into a single Go service since both use Go.

- **Rejected because:** LLM proxying and Kubernetes backup have no shared domain logic. Combining them would create an artificial coupling that complicates deployment (a Kubernetes cluster dependency would be introduced for LLM-only deployments) and scaling (proxy traffic patterns are fundamentally different from backup schedules).

## References

- [The Twelve-Factor App](https://12factor.net/)
- [Microservices vs. Modular Monolith](https://www.thoughtworks.com/insights/blog/microservices-vs-modular-monolith)
- Project README: Architecture section
