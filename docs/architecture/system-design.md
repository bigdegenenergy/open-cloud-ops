# Open Cloud Ops -- System Design

**Version:** 1.0
**Last Updated:** 2026-02-06
**Status:** Living Document

---

## Table of Contents

1. [System Overview](#system-overview)
2. [High-Level Architecture](#high-level-architecture)
3. [Component Descriptions](#component-descriptions)
4. [Data Flow Diagrams](#data-flow-diagrams)
5. [Technology Choices and Rationale](#technology-choices-and-rationale)
6. [Security Architecture](#security-architecture)
7. [Scalability Considerations](#scalability-considerations)
8. [Database Schema Overview](#database-schema-overview)

---

## System Overview

Open Cloud Ops is an open-source AI FinOps platform composed of three independently deployable modules that share a common data layer. The platform provides:

- **LLM cost tracking and budget enforcement** via a reverse proxy gateway
- **Multi-cloud cost optimization** through data ingestion and recommendation engines
- **Kubernetes resilience** with automated backup, recovery, and disaster recovery policies

Each module is designed to run as an independent service, communicating through shared PostgreSQL storage and Redis caching. A unified React dashboard provides a single pane of glass across all three modules.

---

## High-Level Architecture

```
                          +-----------------------+
                          |     React Dashboard   |
                          |   (localhost:3000)     |
                          +----------+------------+
                                     |
                                     | HTTP/REST
                                     v
                    +----------------+----------------+
                    |           API Gateway            |
                    | (routing, auth, rate limiting)    |
                    +----+----------+----------+------+
                         |          |          |
            +------------+    +-----+-----+   +----------+
            |                 |           |               |
            v                 v           v               v
   +--------+------+  +------+-------+  +--------+-------+
   |    Cerebra    |  |   Economist  |  |      Aegis     |
   |  LLM Gateway  |  |  FinOps Core |  | Resilience Eng |
   |  (Go, :8080)  |  | (Py, :8081)  |  |  (Go, :8082)   |
   +---+-----+----+  +---+-----+---+  +---+------+------+
       |     |            |     |          |      |
       |     |            |     |          |      |
       v     v            v     v          v      v
   +---+-----+------------+-----+----------+------+------+
   |                   PostgreSQL + TimescaleDB           |
   |                     (localhost:5432)                  |
   +-----------------------------------------------------+
   |                        Redis                         |
   |                     (localhost:6379)                  |
   +-----------------------------------------------------+

   External Integrations:
   +------------+   +----------+   +----------+   +--------+
   | OpenAI API |   |Anthropic |   |  Gemini  |   |  AWS   |
   |            |   |   API    |   |   API    |   | Azure  |
   +------------+   +----------+   +----------+   | GCP    |
                                                   +--------+
   Observability:
   +-------------------+   +----------+
   | Prometheus + OTel |-->|  Grafana |
   +-------------------+   +----------+
```

---

## Component Descriptions

### Cerebra -- LLM Gateway (Go, Port 8080)

Cerebra is a high-performance reverse proxy that sits between LLM consumers (AI agents, applications, developers) and upstream LLM providers (OpenAI, Anthropic, Google Gemini). It intercepts every request to extract metadata, enforce budgets, and optionally reroute to cheaper models.

**Internal Packages:**

| Package | Path | Responsibility |
|---------|------|----------------|
| `proxy` | `cerebra/internal/proxy/` | Reverse proxy handler -- forwards requests to upstream providers, extracts token counts and cost metadata from responses |
| `router` | `cerebra/internal/router/` | Smart model routing engine -- analyzes query complexity and selects optimal model tier (economy, standard, premium) based on configurable strategies |
| `budget` | `cerebra/internal/budget/` | Budget enforcement -- checks per-agent, per-team, per-user, and per-org spending limits before allowing requests through |
| `analytics` | `cerebra/internal/analytics/` | Insights engine -- detects cost spikes, recommends model switches, generates reports |
| `middleware` | `cerebra/internal/middleware/` | HTTP middleware for authentication, logging, CORS, and rate limiting |

**Public Packages:**

| Package | Path | Responsibility |
|---------|------|----------------|
| `models` | `cerebra/pkg/models/` | Core data structures: `APIRequest`, `Agent`, `Team`, `Organization`, `ModelPricing`, `CostSummary` |
| `config` | `cerebra/pkg/config/` | Environment-based configuration loading (PostgreSQL, Redis, CORS settings) |
| `database` | `cerebra/pkg/database/` | PostgreSQL connection pool (pgx) and schema initialization including TimescaleDB hypertables |
| `cache` | `cerebra/pkg/cache/` | Redis client wrapper for budget caching and rate limiting |

### Economist -- FinOps Core (Python/FastAPI, Port 8081)

Economist is a multi-cloud cost management engine built with Python and FastAPI. It ingests cost data from AWS, Azure, and GCP, analyzes spending patterns, generates optimization recommendations, and enforces governance policies.

**Internal Packages:**

| Package | Path | Responsibility |
|---------|------|----------------|
| `ingestion` | `economist/internal/ingestion/` | Cloud cost data ingestion from AWS Cost Explorer, Azure Cost Management, and GCP Billing APIs |
| `optimizer` | `economist/internal/optimizer/` | Cost optimization engine that identifies idle resources, rightsizing opportunities, spot/reserved instance savings |
| `policy` | `economist/internal/policy/` | Governance policy engine for automated compliance checking and violation tracking |

**Public Packages:**

| Package | Path | Responsibility |
|---------|------|----------------|
| `cloud` | `economist/pkg/cloud/` | Cloud provider integration adapters (AWS boto3, Azure SDK, GCP client libraries) |
| `cost` | `economist/pkg/cost/` | Cost calculation and aggregation utilities |
| `config` | `economist/pkg/config.py` | Pydantic-based settings loaded from environment variables |
| `database` | `economist/pkg/database.py` | SQLAlchemy ORM models and session management |

**ORM Models:**

- `CloudCost` -- individual cloud cost line items with provider, service, resource, region, tags
- `OptimizationRecommendation` -- generated savings recommendations with confidence scores
- `GovernancePolicy` -- cost/resource governance rules with JSONB rule definitions
- `PolicyViolation` -- detected governance violations linked to policies

### Aegis -- Resilience Engine (Go, Port 8082)

Aegis provides Kubernetes-native backup, disaster recovery, and health monitoring capabilities. It orchestrates backup schedules, manages recovery workflows, enforces DR policies, and monitors cluster health.

**Internal Packages:**

| Package | Path | Responsibility |
|---------|------|----------------|
| `backup` | `aegis/internal/backup/` | Backup orchestration -- schedules, executes, and tracks backup jobs via Velero |
| `recovery` | `aegis/internal/recovery/` | Recovery workflow management -- plan creation, execution, and status tracking |
| `policy` | `aegis/internal/policy/` | DR policy engine -- defines and enforces retention rules, RPO/RTO targets, compliance checks |
| `health` | `aegis/internal/health/` | Cluster health monitoring -- aggregates backup status, policy compliance, and recovery readiness |

**Public Packages:**

| Package | Path | Responsibility |
|---------|------|----------------|
| `config` | `aegis/pkg/config/` | Environment-based configuration for Velero, storage backends, and Kubernetes connectivity |
| `models` | `aegis/pkg/models/` | Data structures for backup jobs, recovery plans, DR policies, and health reports |

---

## Data Flow Diagrams

### 1. LLM Request Proxy Flow

This diagram shows how Cerebra handles an incoming LLM API request from an AI agent.

```
  AI Agent / Application
        |
        | POST /api/v1/proxy/openai/v1/chat/completions
        | Headers: Authorization: Bearer <api-key>
        |          X-Agent-ID: agent-123
        |          X-Team-ID: team-456
        v
  +-----+-----+
  |  Cerebra   |
  |  Gateway   |
  +-----+------+
        |
        |  1. Parse request metadata
        |     (provider, model, agent ID, team ID)
        |     NOTE: Request body is NOT inspected or stored
        v
  +-----+------+
  | Budget     |
  | Enforcer   |
  +-----+------+
        |
        |  2. Check budget (Redis cache -> PostgreSQL fallback)
        |     Compare current spend + estimated cost vs. limit
        |
        +--------> [OVER BUDGET] --> Return 429 Too Many Requests
        |                            {"error": "budget_exceeded"}
        |
        |  3. (Optional) Smart Router evaluates model tier
        |     May reroute to cheaper model if strategy allows
        v
  +-----+------+
  | Smart      |
  | Router     |
  +-----+------+
        |
        |  4. Forward request to upstream LLM provider
        |     API key passed through in-memory (never stored)
        v
  +-----+------+        +------------------+
  | HTTP       | -----> | OpenAI /         |
  | Forward    |        | Anthropic /      |
  |            | <----- | Gemini API       |
  +-----+------+        +------------------+
        |
        |  5. Extract response metadata:
        |     - Input tokens, output tokens
        |     - Latency (ms)
        |     - Status code
        |     - Calculate cost from model pricing table
        v
  +-----+------+
  | Metadata   |
  | Logger     |
  +-----+------+
        |
        |  6. Write api_requests record to PostgreSQL
        |     (id, provider, model, agent_id, tokens, cost, timestamp)
        |     NO prompt or response content is stored
        |
        |  7. Update spend in Redis cache + PostgreSQL
        v
  +-----+------+
  | Return     |
  | Response   |
  +-----+------+
        |
        |  Original LLM response returned to caller unmodified
        v
  AI Agent / Application
```

### 2. Cost Data Ingestion Flow (Economist)

This diagram shows how Economist ingests cost data from multiple cloud providers.

```
  Scheduled Trigger (Celery Beat / Cron)
        |
        |  Runs on configurable schedule (e.g., every 6 hours)
        v
  +-----+----------+
  | Ingestion      |
  | Orchestrator   |
  +-----+----------+
        |
        +----------+-----------+
        |          |           |
        v          v           v
  +-----+--+ +----+---+ +-----+--+
  |  AWS    | | Azure  | |  GCP   |
  |  Adapter| | Adapter| | Adapter|
  |  (boto3)| | (SDK)  | | (SDK)  |
  +-----+---+ +----+---+ +-----+--+
        |          |           |
        v          v           v
  AWS Cost    Azure Cost   GCP Billing
  Explorer    Management   Export API
        |          |           |
        +----------+-----------+
        |
        |  Normalized cost line items:
        |  {provider, service, resource_id, cost_usd,
        |   region, account_id, tags, date}
        v
  +-----+----------+
  | Data           |
  | Normalizer     |
  +-----+----------+
        |
        |  Deduplicate, validate, normalize currency
        v
  +-----+----------+
  | PostgreSQL     |
  | cloud_costs    |
  | table          |
  +-----+----------+
        |
        |  Trigger post-ingestion analysis
        v
  +-----+-----------+     +------------------+
  | Optimizer       |---->| optimization_    |
  | Engine          |     | recommendations  |
  +-----+-----------+     +------------------+
        |
        v
  +-----+-----------+     +------------------+
  | Policy          |---->| policy_          |
  | Engine          |     | violations       |
  +-----------------+     +------------------+
```

### 3. Backup and Recovery Flow (Aegis)

This diagram shows how Aegis manages Kubernetes backup and recovery operations.

```
  === BACKUP FLOW ===

  DR Policy Definition
  (schedule, retention, namespaces, RPO/RTO)
        |
        v
  +-----+----------+
  | Policy Engine  |
  | (scheduler)    |
  +-----+----------+
        |
        |  Evaluates cron schedule, triggers backup jobs
        v
  +-----+----------+
  | Backup         |
  | Orchestrator   |
  +-----+----------+
        |
        |  Creates Velero Backup resource
        v
  +-----+----------+     +------------------+
  | Velero         |---->| Object Storage   |
  | (K8s operator) |     | (S3/GCS/Azure    |
  +-----+----------+     |  Blob)           |
        |                 +------------------+
        |
        |  Reports status back
        v
  +-----+----------+
  | Status Tracker |
  | (PostgreSQL)   |
  +----------------+
        |
        |  Updates backup_records table
        |  Checks compliance with DR policies
        v
  +-----+----------+
  | Health         |
  | Monitor        |
  +----------------+


  === RECOVERY FLOW ===

  Recovery Request (API or Dashboard)
        |
        |  POST /api/v1/recovery/plans
        |  {backup_id, target_cluster, namespaces}
        v
  +-----+----------+
  | Recovery       |
  | Planner        |
  +-----+----------+
        |
        |  Validates backup exists, creates recovery plan
        v
  +-----+----------+
  | Recovery Plan  |
  | (pending)      |
  +-----+----------+
        |
        |  POST /api/v1/recovery/execute/{plan_id}
        v
  +-----+----------+
  | Recovery       |
  | Executor       |
  +-----+----------+
        |
        |  Creates Velero Restore resource
        v
  +-----+----------+     +------------------+
  | Velero         |<----| Object Storage   |
  | (K8s operator) |     | (restore from    |
  +-----+----------+     |  backup)         |
        |                 +------------------+
        |
        |  Monitors restore progress
        v
  +-----+----------+
  | Status: OK     |
  | Recovery       |
  | Complete       |
  +----------------+
```

---

## Technology Choices and Rationale

### Language Choices

| Module | Language | Rationale |
|--------|----------|-----------|
| Cerebra | Go 1.21+ | Go provides excellent HTTP performance with minimal memory overhead, critical for a reverse proxy handling high request throughput. The Gin framework offers battle-tested HTTP routing with middleware support. Go's concurrency model (goroutines) is ideal for proxying concurrent LLM requests. |
| Economist | Python 3.11+ (FastAPI) | Python has the richest ecosystem of cloud SDKs (boto3, azure-sdk, google-cloud). FastAPI provides async HTTP handling with automatic OpenAPI documentation. Data analysis libraries (pandas, numpy) are available for cost analytics. |
| Aegis | Go 1.21+ | Go is the native language of the Kubernetes ecosystem. Velero, the underlying backup tool, is written in Go. Using Go enables direct integration with Kubernetes client libraries and CRD management. |

### Data Layer

| Technology | Purpose | Rationale |
|-----------|---------|-----------|
| PostgreSQL 16+ | Primary relational store | Mature, reliable RDBMS with excellent JSON support (JSONB), strong indexing, and extensibility. Handles both relational data (agents, teams, policies) and semi-structured data (tags, rules). |
| TimescaleDB | Time-series extension | The `api_requests` table is a time-series workload (cost per request over time). TimescaleDB hypertables provide automatic partitioning, efficient time-range queries, and continuous aggregates for dashboards. Falls back gracefully to plain PostgreSQL if not installed. |
| Redis 7+ | Cache and queue | Budget lookups must be sub-millisecond to avoid adding latency to LLM requests. Redis provides fast budget caching, rate limiting counters, and serves as a Celery broker for async tasks in Economist. |

### Observability

| Technology | Purpose | Rationale |
|-----------|---------|-----------|
| OpenTelemetry | Distributed tracing and metrics | Vendor-neutral instrumentation standard. Enables tracing requests across Cerebra, Economist, and Aegis without vendor lock-in. |
| Prometheus | Metrics collection | De facto standard for metrics in Kubernetes environments. Pull-based model works well with dynamic container orchestration. |
| Grafana | Visualization | Rich dashboarding with native Prometheus, PostgreSQL, and TimescaleDB data sources. Pre-built panels for cost trends, budget utilization, and backup health. |

### Infrastructure

| Technology | Purpose | Rationale |
|-----------|---------|-----------|
| Docker / Docker Compose | Local development and deployment | Enables reproducible development environments and simple single-node deployments. |
| Kubernetes | Production deployment | Provides auto-scaling, self-healing, and rolling updates for each module independently. Native environment for Aegis backup operations. |
| Velero | Kubernetes backup/restore | Industry-standard open-source tool for Kubernetes cluster backup. Supports multiple storage backends (S3, GCS, Azure Blob). |
| Cloud Custodian | Cloud governance | Rules-as-code engine for enforcing cloud resource policies across AWS, Azure, and GCP. |

---

## Security Architecture

### Core Principle: Privacy-First Design

Open Cloud Ops is built on the principle that **user data privacy is non-negotiable**. This is especially critical for Cerebra, which proxies potentially sensitive LLM prompts and responses.

### API Key Handling

```
  Client                    Cerebra                  LLM Provider
    |                         |                          |
    | Authorization: Bearer   |                          |
    | <user-api-key>          |                          |
    |------------------------>|                          |
    |                         |  Key held in memory only |
    |                         |  NEVER written to:       |
    |                         |  - Database              |
    |                         |  - Logs                  |
    |                         |  - Disk                  |
    |                         |  - Redis                 |
    |                         |                          |
    |                         | Authorization: Bearer    |
    |                         | <user-api-key>           |
    |                         |------------------------->|
    |                         |                          |
    |                         |  Response                |
    |                         |<-------------------------|
    |  Response (unmodified)  |                          |
    |<------------------------|  Key released from       |
    |                         |  memory after request    |
```

### What Is Stored vs. What Is NOT Stored

| Data | Stored? | Location | Purpose |
|------|---------|----------|---------|
| Provider name (openai, anthropic, gemini) | Yes | PostgreSQL | Cost attribution |
| Model name (gpt-4, claude-opus, etc.) | Yes | PostgreSQL | Cost calculation |
| Token counts (input, output, total) | Yes | PostgreSQL | Cost calculation |
| Computed cost (USD) | Yes | PostgreSQL | Budget tracking |
| Latency (ms) | Yes | PostgreSQL | Performance monitoring |
| Agent/Team/Org IDs | Yes | PostgreSQL | Cost allocation |
| Timestamp | Yes | PostgreSQL | Time-series analysis |
| **Prompt content** | **NO** | **Nowhere** | **Never stored** |
| **Response content** | **NO** | **Nowhere** | **Never stored** |
| **API keys** | **NO** | **Nowhere** | **Never stored** |

### Network Security

- All inter-service communication uses internal Docker/Kubernetes networking
- External-facing endpoints are exposed through a single API gateway
- CORS is configurable via `CEREBRA_ALLOWED_ORIGINS` environment variable
- Redis and PostgreSQL are not exposed to external networks in production deployments
- TLS termination occurs at the ingress/load balancer level

### Authentication and Authorization

- API key passthrough: Cerebra does not manage its own API keys for LLM providers; users supply their own
- Service-level authentication is handled via API gateway tokens
- RBAC is planned for Phase 4 (enterprise features)

---

## Scalability Considerations

### Horizontal Scaling

Each module can be scaled independently based on its specific load characteristics:

```
                    Load Balancer
                         |
          +--------------+--------------+
          |              |              |
     +----+----+    +----+----+    +----+----+
     | Cerebra |    | Cerebra |    | Cerebra |
     |  Pod 1  |    |  Pod 2  |    |  Pod N  |
     +---------+    +---------+    +---------+
          |              |              |
          +--------------+--------------+
                         |
                  +------+------+
                  | PostgreSQL  |
                  | (primary)   |
                  +------+------+
                         |
                  +------+------+
                  |   Read      |
                  |  Replicas   |
                  +-------------+
```

- **Cerebra** scales horizontally to handle increased LLM proxy throughput. Each instance is stateless; all state lives in PostgreSQL and Redis.
- **Economist** scales horizontally for API serving. Ingestion workers scale as Celery workers behind Redis.
- **Aegis** typically runs fewer replicas since backup operations are scheduled and not latency-sensitive.

### Database Scaling

- **TimescaleDB hypertables** automatically partition the `api_requests` table by time, enabling efficient pruning of old data and parallel query execution
- **Read replicas** can be added for analytics queries without impacting write performance
- **Connection pooling** is configured at the application level (pgx pool for Go, SQLAlchemy pool for Python) with sensible defaults (max 20 connections for Cerebra, pool_size=10 + max_overflow=20 for Economist)
- **Redis** serves as a write-through cache for budget data, reducing database reads on the hot path

### Performance Targets

| Metric | Target | Notes |
|--------|--------|-------|
| Proxy latency overhead | < 5ms | Time added by Cerebra on top of upstream LLM latency |
| Budget check latency | < 1ms | Redis cache hit path |
| Cost ingestion throughput | 100K records/hour | Per Economist worker |
| Backup initiation time | < 30s | Time from trigger to Velero job creation |

---

## Database Schema Overview

### Cerebra Schema (PostgreSQL + TimescaleDB)

The Cerebra schema is initialized automatically on startup via `database.go`. All tables use `CREATE TABLE IF NOT EXISTS` for idempotent initialization.

```sql
-- Organizational hierarchy
organizations (id TEXT PK, name TEXT, created_at TIMESTAMPTZ)
    |
    +-- teams (id TEXT PK, name TEXT, org_id TEXT FK, created_at TIMESTAMPTZ)
           |
           +-- agents (id TEXT PK, name TEXT, team_id TEXT FK, tags TEXT[], created_at TIMESTAMPTZ)

-- LLM model pricing reference
model_pricing (
    provider TEXT,
    model TEXT,
    input_per_m_token DOUBLE PRECISION,   -- Cost per 1M input tokens
    output_per_m_token DOUBLE PRECISION,  -- Cost per 1M output tokens
    updated_at TIMESTAMPTZ,
    PRIMARY KEY (provider, model)
)

-- Budget controls
budgets (
    id TEXT PK,
    scope TEXT,          -- 'agent', 'team', 'user', 'org'
    entity_id TEXT,
    limit_usd DOUBLE PRECISION,
    spent_usd DOUBLE PRECISION,
    period BIGINT,       -- Duration in nanoseconds
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    UNIQUE(scope, entity_id)
)

-- API request log (TimescaleDB hypertable)
api_requests (
    id TEXT,
    provider TEXT,
    model TEXT,
    agent_id TEXT,
    team_id TEXT,
    org_id TEXT,
    input_tokens BIGINT,
    output_tokens BIGINT,
    total_tokens BIGINT,
    cost_usd DOUBLE PRECISION,
    latency_ms BIGINT,
    status_code INTEGER,
    was_routed BOOLEAN,
    original_model TEXT,
    routed_model TEXT,
    savings_usd DOUBLE PRECISION,
    timestamp TIMESTAMPTZ          -- Hypertable partition key
)
-- Converted to TimescaleDB hypertable on 'timestamp' column
```

**Indexes:**

| Index | Columns | Purpose |
|-------|---------|---------|
| `idx_api_requests_timestamp` | `timestamp DESC` | Time-range queries for dashboards |
| `idx_api_requests_agent_id` | `agent_id, timestamp DESC` | Per-agent cost lookup |
| `idx_api_requests_team_id` | `team_id, timestamp DESC` | Per-team cost lookup |
| `idx_api_requests_org_id` | `org_id, timestamp DESC` | Per-org cost lookup |
| `idx_api_requests_model` | `model, timestamp DESC` | Per-model analytics |
| `idx_api_requests_provider` | `provider, timestamp DESC` | Per-provider analytics |
| `idx_budgets_scope_entity` | `scope, entity_id` | Fast budget lookups |

### Economist Schema (SQLAlchemy ORM)

The Economist schema is managed through SQLAlchemy ORM models with automatic table creation.

```sql
-- Cloud cost line items
cloud_costs (
    id UUID PK,
    provider VARCHAR(32),
    service VARCHAR(128),
    resource_id VARCHAR(256),
    resource_name VARCHAR(256),
    cost_usd FLOAT,
    currency VARCHAR(8),
    usage_quantity FLOAT,
    usage_unit VARCHAR(64),
    region VARCHAR(64),
    account_id VARCHAR(128),
    tags JSONB,
    date DATE,
    created_at TIMESTAMPTZ
)

-- Optimization recommendations
optimization_recommendations (
    id UUID PK,
    provider VARCHAR(32),
    resource_id VARCHAR(256),
    resource_type VARCHAR(128),
    recommendation_type VARCHAR(64),
    title VARCHAR(512),
    description TEXT,
    estimated_monthly_savings FLOAT,
    confidence FLOAT,
    status VARCHAR(32),          -- 'open', 'accepted', 'dismissed'
    created_at TIMESTAMPTZ,
    resolved_at TIMESTAMPTZ
)

-- Governance policies
governance_policies (
    id UUID PK,
    name VARCHAR(256) UNIQUE,
    description TEXT,
    policy_type VARCHAR(64),
    rules JSONB,                 -- Flexible rule definitions
    severity VARCHAR(32),        -- 'info', 'warning', 'critical'
    enabled BOOLEAN,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
)

-- Policy violations
policy_violations (
    id UUID PK,
    policy_id UUID,
    resource_id VARCHAR(256),
    provider VARCHAR(32),
    description TEXT,
    severity VARCHAR(32),
    detected_at TIMESTAMPTZ,
    resolved_at TIMESTAMPTZ
)
```

**Composite Indexes:**

| Index | Columns | Purpose |
|-------|---------|---------|
| `ix_cloud_costs_provider_date` | `provider, date` | Filter costs by provider over time |
| `ix_cloud_costs_service_date` | `service, date` | Filter costs by service over time |

### Entity Relationship Diagram

```
  Cerebra Domain:
  +-----------------+     +-----------------+     +-----------------+
  | organizations   |---->|     teams       |---->|     agents      |
  | (id, name)      | 1:N | (id, org_id)    | 1:N | (id, team_id)   |
  +-----------------+     +-----------------+     +--------+--------+
                                                           |
                                                           | 1:N
                                                           v
  +-----------------+     +-----------------+     +--------+--------+
  | model_pricing   |     |    budgets      |     |  api_requests   |
  | (provider,model)|     | (scope,entity)  |     | (agent, model,  |
  +-----------------+     +-----------------+     |  cost, tokens)  |
                                                  +-----------------+

  Economist Domain:
  +-----------------+     +----------------------+
  | cloud_costs     |     | optimization_        |
  | (provider,      |     | recommendations      |
  |  service, date) |     | (resource, savings)  |
  +-----------------+     +----------------------+

  +-----------------+     +----------------------+
  | governance_     |---->| policy_violations    |
  | policies        | 1:N | (policy_id,          |
  | (rules, type)   |     |  resource_id)        |
  +-----------------+     +----------------------+
```

---

## Appendix: Configuration Reference

### Cerebra Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CEREBRA_PORT` | `8080` | HTTP server port |
| `CEREBRA_LOG_LEVEL` | `info` | Log verbosity (debug, info, warn, error) |
| `CEREBRA_ALLOWED_ORIGINS` | `http://localhost:3000` | Comma-separated CORS origins |
| `POSTGRES_HOST` | `localhost` | PostgreSQL host |
| `POSTGRES_PORT` | `5432` | PostgreSQL port |
| `POSTGRES_DB` | `cerebra` | Database name |
| `POSTGRES_USER` | `cerebra` | Database user |
| `POSTGRES_PASSWORD` | (empty) | Database password |
| `DATABASE_URL` | (derived) | Full PostgreSQL URL (overrides individual settings) |
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_URL` | (derived) | Full Redis URL (overrides individual settings) |

### Economist Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ECONOMIST_PORT` | `8081` | HTTP server port |
| `LOG_LEVEL` | `INFO` | Log verbosity |
| `POSTGRES_HOST` | `localhost` | PostgreSQL host |
| `POSTGRES_PORT` | `5432` | PostgreSQL port |
| `POSTGRES_DB` | `opencloudops` | Database name |
| `POSTGRES_USER` | `oco_user` | Database user |
| `POSTGRES_PASSWORD` | `change_me` | Database password |
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `AWS_ACCESS_KEY_ID` | (empty) | AWS credentials |
| `AWS_SECRET_ACCESS_KEY` | (empty) | AWS credentials |
| `AWS_REGION` | `us-east-1` | AWS region |
| `AZURE_SUBSCRIPTION_ID` | (empty) | Azure subscription |
| `AZURE_TENANT_ID` | (empty) | Azure tenant |
| `AZURE_CLIENT_ID` | (empty) | Azure client ID |
| `AZURE_CLIENT_SECRET` | (empty) | Azure client secret |
| `GCP_PROJECT_ID` | (empty) | GCP project |
| `GCP_CREDENTIALS_JSON` | (empty) | GCP service account JSON |
