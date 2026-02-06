# Quick Start Guide

Get Open Cloud Ops running in under 10 minutes.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Clone and Configure](#clone-and-configure)
3. [Start with Docker Compose](#start-with-docker-compose)
4. [Verify Services Are Running](#verify-services-are-running)
5. [Make Your First API Calls](#make-your-first-api-calls)
6. [View Metrics in Grafana](#view-metrics-in-grafana)
7. [Next Steps](#next-steps)

---

## Prerequisites

Before you begin, make sure you have the following installed:

| Tool | Minimum Version | Check Command | Install Guide |
|------|----------------|---------------|---------------|
| Docker | 24.0+ | `docker --version` | [docs.docker.com](https://docs.docker.com/get-docker/) |
| Docker Compose | 2.20+ (V2) | `docker compose version` | Included with Docker Desktop |
| Git | 2.30+ | `git --version` | [git-scm.com](https://git-scm.com/downloads) |
| curl | Any | `curl --version` | Pre-installed on most systems |

**Optional** (for local development without Docker):

| Tool | Minimum Version | Purpose |
|------|----------------|---------|
| Go | 1.21+ | Running Cerebra and Aegis locally |
| Python | 3.11+ | Running Economist locally |
| Node.js | 22+ | Running the React dashboard locally |
| pnpm | 8+ | Package manager for the dashboard |

---

## Clone and Configure

### Step 1: Clone the Repository

```bash
git clone https://github.com/bigdegenenergy/open-cloud-ops.git
cd open-cloud-ops
```

### Step 2: Create Your Environment File

Copy the example environment file and customize it:

```bash
cp .env.example .env
```

Edit `.env` with your settings. At minimum, you should set a strong database password:

```bash
# .env - Minimum required configuration

# PostgreSQL
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_DB=opencloudops
POSTGRES_USER=oco_user
POSTGRES_PASSWORD=your-secure-password-here

# Redis
REDIS_HOST=redis
REDIS_PORT=6379

# Service Ports
CEREBRA_PORT=8080
ECONOMIST_PORT=8081
AEGIS_PORT=8082
```

**Optional:** If you want Economist to connect to your cloud accounts for cost data, add your cloud credentials:

```bash
# AWS (optional)
AWS_ACCESS_KEY_ID=AKIA...
AWS_SECRET_ACCESS_KEY=...
AWS_REGION=us-east-1

# Azure (optional)
AZURE_SUBSCRIPTION_ID=...
AZURE_TENANT_ID=...
AZURE_CLIENT_ID=...
AZURE_CLIENT_SECRET=...

# GCP (optional)
GCP_PROJECT_ID=my-project
GCP_CREDENTIALS_JSON=...
```

---

## Start with Docker Compose

### Start All Services

```bash
docker compose up -d
```

This starts the following containers:

| Container | Service | Port | Description |
|-----------|---------|------|-------------|
| `oco-postgres` | PostgreSQL + TimescaleDB | 5432 | Primary database |
| `oco-redis` | Redis | 6379 | Cache and queue |
| `oco-cerebra` | Cerebra | 8080 | LLM Gateway |
| `oco-economist` | Economist | 8081 | FinOps Core |
| `oco-aegis` | Aegis | 8082 | Resilience Engine |
| `oco-prometheus` | Prometheus | 9090 | Metrics collection |
| `oco-grafana` | Grafana | 3001 | Metrics dashboards |

### Check Container Status

```bash
docker compose ps
```

Expected output:

```
NAME             IMAGE                    STATUS          PORTS
oco-postgres     timescale/timescaledb    Up (healthy)    0.0.0.0:5432->5432/tcp
oco-redis        redis:7-alpine           Up (healthy)    0.0.0.0:6379->6379/tcp
oco-cerebra      oco/cerebra:latest       Up              0.0.0.0:8080->8080/tcp
oco-economist    oco/economist:latest     Up              0.0.0.0:8081->8081/tcp
oco-aegis        oco/aegis:latest         Up              0.0.0.0:8082->8082/tcp
oco-prometheus   prom/prometheus          Up              0.0.0.0:9090->9090/tcp
oco-grafana      grafana/grafana          Up              0.0.0.0:3001->3001/tcp
```

### View Logs

To see the logs for all services:

```bash
docker compose logs -f
```

To see logs for a specific service:

```bash
docker compose logs -f cerebra
docker compose logs -f economist
docker compose logs -f aegis
```

---

## Verify Services Are Running

Run health checks against each service to confirm they are operational.

### Check Cerebra (LLM Gateway)

```bash
curl -s http://localhost:8080/health | python3 -m json.tool
```

Expected output:

```json
{
    "status": "healthy",
    "service": "cerebra",
    "version": "0.1.0",
    "dependencies": {
        "postgresql": "connected",
        "redis": "connected"
    }
}
```

### Check Economist (FinOps Core)

```bash
curl -s http://localhost:8081/health | python3 -m json.tool
```

Expected output:

```json
{
    "status": "healthy",
    "service": "economist"
}
```

### Check Aegis (Resilience Engine)

```bash
curl -s http://localhost:8082/health | python3 -m json.tool
```

Expected output:

```json
{
    "status": "healthy",
    "service": "aegis",
    "version": "0.1.0",
    "dependencies": {
        "postgresql": "connected",
        "redis": "connected",
        "velero": "connected",
        "kubernetes": "connected"
    }
}
```

### Quick Health Check Script

You can check all services at once:

```bash
echo "=== Cerebra ===" && curl -sf http://localhost:8080/health && echo ""
echo "=== Economist ===" && curl -sf http://localhost:8081/health && echo ""
echo "=== Aegis ===" && curl -sf http://localhost:8082/health && echo ""
```

---

## Make Your First API Calls

### 1. Proxy an LLM Request Through Cerebra

Send a request to OpenAI through the Cerebra proxy. Replace `sk-your-openai-key` with your actual OpenAI API key.

```bash
curl -X POST http://localhost:8080/api/v1/proxy/openai/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-openai-key" \
  -H "X-Agent-ID: my-first-agent" \
  -H "X-Team-ID: my-team" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [
      {"role": "user", "content": "Say hello in exactly 5 words."}
    ],
    "max_tokens": 50
  }'
```

The response comes from OpenAI, but Cerebra has logged the cost metadata. You can verify this by checking the analytics:

```bash
curl -s http://localhost:8080/api/v1/analytics/summary?group_by=agent | python3 -m json.tool
```

### 2. Set a Budget for Your Agent

Create a $10/month budget for your test agent:

```bash
curl -X POST http://localhost:8080/api/v1/budgets \
  -H "Content-Type: application/json" \
  -d '{
    "scope": "agent",
    "entity_id": "my-first-agent",
    "limit_usd": 10.00,
    "period": "monthly"
  }'
```

### 3. Check Cloud Cost Summary

Query the Economist module for cloud costs (this requires cloud provider credentials to be configured):

```bash
curl -s http://localhost:8081/api/v1/costs/summary | python3 -m json.tool
```

### 4. List Governance Policies

```bash
curl -s http://localhost:8081/api/v1/governance/policies | python3 -m json.tool
```

### 5. Create a Governance Policy

Create a tag enforcement policy:

```bash
curl -X POST http://localhost:8081/api/v1/governance/policies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Require Cost Center Tag",
    "description": "All resources must have a cost-center tag.",
    "policy_type": "tag_enforcement",
    "rules": {
      "required_tags": ["cost-center"],
      "applies_to": ["aws", "azure", "gcp"]
    },
    "severity": "warning",
    "enabled": true
  }'
```

### 6. Check Backup Health (Aegis)

```bash
curl -s http://localhost:8082/api/v1/health/summary | python3 -m json.tool
```

### 7. Create a DR Policy

Create a daily backup policy (requires a Kubernetes cluster with Velero installed):

```bash
curl -X POST http://localhost:8082/api/v1/dr/policies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "dev-daily-backup",
    "description": "Daily backup of the dev cluster.",
    "cluster": "dev-local",
    "namespaces": ["default"],
    "schedule": "0 2 * * *",
    "retention_days": 7,
    "include_volumes": false,
    "storage_backend": "s3",
    "storage_location": "s3://my-backups/dev/",
    "rpo_hours": 24,
    "rto_hours": 4,
    "enabled": true
  }'
```

---

## View Metrics in Grafana

### Access Grafana

Open your browser and navigate to:

```
http://localhost:3001
```

Default credentials:

- **Username:** `admin`
- **Password:** `admin` (you will be prompted to change this on first login)

### Pre-Configured Dashboards

After logging in, navigate to **Dashboards** in the left sidebar. The following dashboards are pre-configured:

1. **Open Cloud Ops Overview** -- Cross-module summary with total LLM spend, cloud costs, and backup health
2. **Cerebra - LLM Gateway** -- Real-time LLM API costs, request rates, latency percentiles, budget utilization, and routing savings
3. **Economist - Cloud FinOps** -- Multi-cloud cost trends, top services, optimization savings, and policy violations
4. **Aegis - Resilience** -- Backup success rates, storage consumption, RPO/RTO compliance, and cluster health

### Adding Prometheus as a Data Source (Manual Setup)

If Prometheus is not pre-configured as a data source:

1. Go to **Configuration** (gear icon) -> **Data Sources**
2. Click **Add data source**
3. Select **Prometheus**
4. Set URL to `http://prometheus:9090` (Docker internal) or `http://localhost:9090` (if accessing externally)
5. Click **Save & Test**

### Key Metrics to Monitor

| Metric | Source | Dashboard |
|--------|--------|-----------|
| `cerebra_proxy_requests_total` | Cerebra | LLM Gateway |
| `cerebra_proxy_cost_usd_total` | Cerebra | LLM Gateway |
| `cerebra_proxy_latency_ms` | Cerebra | LLM Gateway |
| `cerebra_budget_utilization_pct` | Cerebra | LLM Gateway |
| `economist_cloud_cost_usd` | Economist | Cloud FinOps |
| `economist_recommendations_total` | Economist | Cloud FinOps |
| `aegis_backup_success_total` | Aegis | Resilience |
| `aegis_backup_duration_seconds` | Aegis | Resilience |
| `aegis_rpo_compliance` | Aegis | Resilience |

---

## Next Steps

Now that you have Open Cloud Ops running, here are some suggested next steps:

1. **Read the API Reference docs** for detailed endpoint documentation:
   - [Cerebra API Reference](../api-reference/cerebra-api.md)
   - [Economist API Reference](../api-reference/economist-api.md)
   - [Aegis API Reference](../api-reference/aegis-api.md)

2. **Configure cloud provider credentials** to enable multi-cloud cost ingestion in Economist.

3. **Set up budget alerts** in Cerebra for your AI agents and teams.

4. **Install Velero** in your Kubernetes cluster to enable Aegis backup/recovery functionality.

5. **Read the Development Guide** if you want to contribute: [Development Guide](./development.md)

6. **Explore the architecture** to understand how the modules interact: [System Design](../architecture/system-design.md)

---

## Stopping and Cleaning Up

### Stop all services (preserves data):

```bash
docker compose stop
```

### Stop and remove containers (preserves volumes/data):

```bash
docker compose down
```

### Stop and remove everything including data volumes:

```bash
docker compose down -v
```

---

## Troubleshooting

### Service fails to start

Check the logs for the specific service:

```bash
docker compose logs cerebra
docker compose logs economist
docker compose logs aegis
```

### Database connection refused

Ensure PostgreSQL is running and healthy:

```bash
docker compose ps postgres
docker compose logs postgres
```

The database may take a few seconds to initialize on first startup. Services will retry connections automatically.

### Port already in use

If a port is already in use on your machine, modify the port mapping in `docker-compose.yml` or your `.env` file:

```bash
# Example: Change Cerebra to port 9080
CEREBRA_PORT=9080
```

### Redis connection issues

```bash
docker compose logs redis
docker exec oco-redis redis-cli ping
```

Expected response: `PONG`
