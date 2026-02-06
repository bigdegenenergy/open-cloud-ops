# Development Guide

This guide covers everything you need to set up a local development environment, run services, execute tests, and contribute to Open Cloud Ops.

---

## Table of Contents

1. [Setting Up the Development Environment](#setting-up-the-development-environment)
2. [Running Services Locally](#running-services-locally)
3. [Running Tests](#running-tests)
4. [Code Style and Conventions](#code-style-and-conventions)
5. [Project Structure](#project-structure)
6. [Database Development](#database-development)
7. [Contributing Workflow](#contributing-workflow)
8. [Common Development Tasks](#common-development-tasks)

---

## Setting Up the Development Environment

### Prerequisites

Install the following tools on your development machine:

| Tool | Version | Purpose | Install |
|------|---------|---------|---------|
| Go | 1.21+ | Cerebra and Aegis services | [go.dev/dl](https://go.dev/dl/) |
| Python | 3.11+ | Economist service | [python.org](https://www.python.org/downloads/) |
| Node.js | 22+ | React dashboard | [nodejs.org](https://nodejs.org/) |
| pnpm | 8+ | Dashboard package manager | `npm install -g pnpm` |
| Docker | 24+ | Infrastructure services | [docker.com](https://docs.docker.com/get-docker/) |
| Docker Compose | 2.20+ | Multi-container orchestration | Included with Docker Desktop |
| Git | 2.30+ | Version control | [git-scm.com](https://git-scm.com/) |

**Optional but recommended:**

| Tool | Purpose | Install |
|------|---------|---------|
| `golangci-lint` | Go linting | `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` |
| `black` | Python formatting | `pip install black` |
| `ruff` | Python linting | `pip install ruff` |
| `pre-commit` | Git hook management | `pip install pre-commit` |

### Step 1: Clone the Repository

```bash
git clone https://github.com/bigdegenenergy/open-cloud-ops.git
cd open-cloud-ops
```

### Step 2: Start Infrastructure Services

Start PostgreSQL (with TimescaleDB) and Redis using Docker Compose:

```bash
docker compose -f deploy/docker/docker-compose.dev.yml up -d
```

This starts only the infrastructure dependencies, not the application services, so you can run those locally for development with hot-reloading.

Verify the infrastructure is running:

```bash
# Check PostgreSQL
docker exec oco-postgres pg_isready -U oco_user
# Expected: accepting connections

# Check Redis
docker exec oco-redis redis-cli ping
# Expected: PONG
```

### Step 3: Configure Environment

Copy the example environment file:

```bash
cp .env.example .env
```

For local development, the defaults should work. The key settings are:

```bash
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=opencloudops
POSTGRES_USER=oco_user
POSTGRES_PASSWORD=change_me

REDIS_HOST=localhost
REDIS_PORT=6379
```

### Step 4: Install Dependencies

Install dependencies for each module you plan to work on:

```bash
# Cerebra (Go)
cd cerebra && go mod download && cd ..

# Aegis (Go)
cd aegis && go mod download && cd ..

# Economist (Python) - use a virtual environment
cd economist
python3 -m venv .venv
source .venv/bin/activate    # On Windows: .venv\Scripts\activate
pip install -r requirements.txt
cd ..

# Dashboard (Node.js)
cd cerebra/web/dashboard
pnpm install
cd ../../..
```

### Step 5: Set Up Pre-Commit Hooks (Optional)

```bash
pre-commit install
```

This runs formatting and linting checks automatically before each commit.

---

## Running Services Locally

### Running Cerebra (LLM Gateway)

```bash
cd cerebra
go run cmd/main.go
```

Output:

```
==============================================
  Cerebra - Open Cloud Ops LLM Gateway
==============================================
Starting server on port 8080...
Cerebra LLM Gateway is ready on :8080
```

**With environment overrides:**

```bash
CEREBRA_PORT=9080 CEREBRA_LOG_LEVEL=debug go run cmd/main.go
```

**With hot-reloading** (using [air](https://github.com/cosmtrek/air)):

```bash
# Install air
go install github.com/cosmtrek/air@latest

# Run with auto-reload
cd cerebra
air
```

### Running Economist (FinOps Core)

```bash
cd economist
source .venv/bin/activate
uvicorn cmd.main:app --reload --port 8081
```

The `--reload` flag enables auto-reloading when Python files change.

Output:

```
==============================================
  Economist - Open Cloud Ops FinOps Core
==============================================
INFO:     Uvicorn running on http://0.0.0.0:8081 (Press CTRL+C to quit)
INFO:     Started reloader process
```

**FastAPI automatic API documentation** is available at:

- Swagger UI: `http://localhost:8081/docs`
- ReDoc: `http://localhost:8081/redoc`

### Running Aegis (Resilience Engine)

```bash
cd aegis
go run cmd/main.go
```

**Note:** Aegis requires a Kubernetes cluster and Velero installation for full functionality. For development without Kubernetes, the service starts and serves the health endpoint, but backup/recovery operations will fail.

### Running the React Dashboard

```bash
cd cerebra/web/dashboard
pnpm dev
```

The dashboard is available at `http://localhost:3000`.

### Running All Services Together

For convenience, you can run all services in separate terminal tabs or use a process manager. Here is an example using a simple shell script:

```bash
#!/bin/bash
# scripts/dev.sh - Run all services for local development

# Start infrastructure
docker compose -f deploy/docker/docker-compose.dev.yml up -d

# Wait for infrastructure
echo "Waiting for PostgreSQL..."
until docker exec oco-postgres pg_isready -U oco_user 2>/dev/null; do sleep 1; done
echo "PostgreSQL is ready."

echo "Starting services..."

# Start Cerebra in background
(cd cerebra && go run cmd/main.go) &
CEREBRA_PID=$!

# Start Economist in background
(cd economist && source .venv/bin/activate && uvicorn cmd.main:app --reload --port 8081) &
ECONOMIST_PID=$!

# Start Aegis in background
(cd aegis && go run cmd/main.go) &
AEGIS_PID=$!

echo "All services started."
echo "  Cerebra:   http://localhost:8080  (PID: $CEREBRA_PID)"
echo "  Economist: http://localhost:8081  (PID: $ECONOMIST_PID)"
echo "  Aegis:     http://localhost:8082  (PID: $AEGIS_PID)"

# Wait for Ctrl+C and clean up
trap "kill $CEREBRA_PID $ECONOMIST_PID $AEGIS_PID 2>/dev/null; exit" INT
wait
```

---

## Running Tests

### Cerebra Tests (Go)

```bash
# Run all tests
cd cerebra
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test -v ./internal/budget/...
go test -v ./internal/proxy/...
go test -v ./internal/analytics/...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run tests with race detection
go test -race ./...
```

### Aegis Tests (Go)

```bash
cd aegis
go test ./...
go test -v -race -coverprofile=coverage.out ./...
```

### Economist Tests (Python)

```bash
cd economist
source .venv/bin/activate

# Run all tests
pytest

# Run with verbose output
pytest -v

# Run specific test file
pytest tests/test_costs.py

# Run with coverage
pytest --cov=. --cov-report=html

# Run with coverage and minimum threshold
pytest --cov=. --cov-fail-under=80
```

### Dashboard Tests (JavaScript)

```bash
cd cerebra/web/dashboard

# Run tests
pnpm test

# Run tests in watch mode
pnpm test -- --watch

# Run with coverage
pnpm test -- --coverage
```

### Integration Tests

Integration tests require the infrastructure services (PostgreSQL, Redis) to be running:

```bash
# Start infrastructure
docker compose -f deploy/docker/docker-compose.dev.yml up -d

# Run integration tests (Go)
cd cerebra
go test -tags=integration ./...

# Run integration tests (Python)
cd economist
pytest -m integration
```

### Running All Tests

```bash
# scripts/test-all.sh
#!/bin/bash
set -e

echo "=== Cerebra Tests ==="
(cd cerebra && go test -race ./...)

echo "=== Aegis Tests ==="
(cd aegis && go test -race ./...)

echo "=== Economist Tests ==="
(cd economist && source .venv/bin/activate && pytest -v)

echo "=== All tests passed ==="
```

---

## Code Style and Conventions

### Go (Cerebra, Aegis)

**Formatting:**

- Use `gofmt` for formatting (enforced automatically by Go)
- Use `goimports` for import organization
- Run `golangci-lint run` before committing

```bash
# Format code
gofmt -w .

# Run linter
golangci-lint run ./...
```

**Conventions:**

- Follow standard Go project layout: `cmd/`, `internal/`, `pkg/`
- `internal/` packages are private to the module and cannot be imported by other modules
- `pkg/` packages are public and may be imported by other modules or external consumers
- Use table-driven tests
- Error messages start with lowercase and do not end with punctuation
- Context (`context.Context`) is always the first parameter in functions that accept it
- Use structured logging (avoid `fmt.Println` in production code)

**Example test style:**

```go
func TestCheckBudget(t *testing.T) {
    tests := []struct {
        name          string
        scope         BudgetScope
        entityID      string
        estimatedCost float64
        wantAllowed   bool
        wantErr       bool
    }{
        {
            name:          "within budget",
            scope:         ScopeAgent,
            entityID:      "agent-001",
            estimatedCost: 1.00,
            wantAllowed:   true,
            wantErr:       false,
        },
        {
            name:          "exceeds budget",
            scope:         ScopeAgent,
            entityID:      "agent-001",
            estimatedCost: 1000.00,
            wantAllowed:   false,
            wantErr:       false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            e := NewEnforcer()
            got, err := e.CheckBudget(tt.scope, tt.entityID, tt.estimatedCost)
            if (err != nil) != tt.wantErr {
                t.Errorf("CheckBudget() error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.wantAllowed {
                t.Errorf("CheckBudget() = %v, want %v", got, tt.wantAllowed)
            }
        })
    }
}
```

### Python (Economist)

**Formatting and linting:**

- Use `black` for formatting (line length: 88)
- Use `ruff` for linting
- Use type hints for all function signatures

```bash
# Format code
black .

# Run linter
ruff check .

# Auto-fix linting issues
ruff check --fix .
```

**Conventions:**

- Follow PEP 8 naming conventions
- Use `pydantic` models for request/response validation
- Use `async def` for FastAPI route handlers
- Use SQLAlchemy ORM for database operations
- Place configuration in `pkg/config.py` using `pydantic-settings`
- Each module in `internal/` should have its own `__init__.py` with public exports

**Example test style:**

```python
import pytest
from httpx import AsyncClient

from cmd.main import app


@pytest.mark.asyncio
async def test_health_check():
    async with AsyncClient(app=app, base_url="http://test") as client:
        response = await client.get("/health")
    assert response.status_code == 200
    data = response.json()
    assert data["status"] == "healthy"
    assert data["service"] == "economist"


@pytest.mark.asyncio
async def test_cost_summary_returns_providers():
    async with AsyncClient(app=app, base_url="http://test") as client:
        response = await client.get("/api/v1/costs/summary")
    assert response.status_code == 200
    data = response.json()
    assert "providers" in data
```

### TypeScript/React (Dashboard)

**Formatting and linting:**

- Use Prettier for formatting
- Use ESLint for linting
- Use TypeScript strict mode

```bash
# Format code
pnpm prettier --write .

# Run linter
pnpm lint

# Fix linting issues
pnpm lint --fix
```

### Commit Message Convention

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types:**

| Type | Description |
|------|-------------|
| `feat` | A new feature |
| `fix` | A bug fix |
| `docs` | Documentation changes |
| `style` | Code formatting (no logic change) |
| `refactor` | Code refactoring (no feature/fix) |
| `test` | Adding or updating tests |
| `chore` | Build, CI, or dependency changes |

**Scopes:** `cerebra`, `economist`, `aegis`, `dashboard`, `deploy`, `docs`

**Examples:**

```
feat(cerebra): add budget enforcement for per-agent limits
fix(economist): correct cost aggregation for multi-currency accounts
docs(aegis): add backup retention policy examples
test(cerebra): add integration tests for proxy handler
chore(deploy): update Docker base images to latest
```

---

## Project Structure

```
open-cloud-ops/
├── cerebra/                     # LLM Gateway (Go)
│   ├── cmd/main.go              # Application entrypoint
│   ├── internal/                # Private packages
│   │   ├── proxy/handler.go     # Reverse proxy for LLM providers
│   │   ├── router/router.go     # Smart model routing engine
│   │   ├── budget/enforcer.go   # Budget enforcement
│   │   ├── analytics/insights.go # Cost insights engine
│   │   └── middleware/          # HTTP middleware
│   ├── pkg/                     # Public packages
│   │   ├── models/models.go     # Core data structures
│   │   ├── config/config.go     # Configuration management
│   │   ├── database/database.go # PostgreSQL connection + schema
│   │   ├── cache/               # Redis client
│   │   └── utils/               # Shared utilities
│   ├── api/                     # OpenAPI definitions
│   ├── web/dashboard/           # React frontend
│   └── go.mod                   # Go module definition
│
├── economist/                   # FinOps Core (Python/FastAPI)
│   ├── cmd/main.py              # FastAPI application + routes
│   ├── internal/                # Private packages
│   │   ├── ingestion/           # Cloud cost data ingestion
│   │   ├── optimizer/           # Cost optimization engine
│   │   └── policy/              # Governance policy engine
│   ├── pkg/                     # Public packages
│   │   ├── config.py            # Pydantic-settings configuration
│   │   ├── database.py          # SQLAlchemy ORM models + sessions
│   │   ├── cloud/               # Cloud provider adapters
│   │   └── cost/                # Cost calculation utilities
│   ├── api/                     # API router definitions
│   └── requirements.txt         # Python dependencies
│
├── aegis/                       # Resilience Engine (Go)
│   ├── cmd/                     # Application entrypoint
│   ├── internal/                # Private packages
│   │   ├── backup/              # Backup orchestration
│   │   ├── recovery/            # Recovery workflows
│   │   ├── policy/              # DR policy engine
│   │   └── health/              # Health monitoring
│   ├── pkg/                     # Public packages
│   │   ├── config/              # Configuration
│   │   └── models/              # Data structures
│   └── go.mod                   # Go module definition
│
├── deploy/                      # Deployment configurations
│   ├── docker/                  # Docker Compose files
│   └── kubernetes/              # Helm charts, K8s manifests
│
├── scripts/                     # Build, test, utility scripts
├── docs/                        # Documentation
│   ├── architecture/            # System design, ADRs
│   ├── api-reference/           # API documentation
│   └── guides/                  # User and developer guides
│
├── .env.example                 # Example environment variables
├── CONTRIBUTING.md              # Contribution guidelines
├── LICENSE                      # Apache 2.0
└── README.md                    # Project overview
```

---

## Database Development

### Accessing the Database

```bash
# Connect to PostgreSQL via Docker
docker exec -it oco-postgres psql -U oco_user -d opencloudops

# Or use psql directly if installed locally
psql -h localhost -p 5432 -U oco_user -d opencloudops
```

### Cerebra Schema

Cerebra manages its own schema initialization in `cerebra/pkg/database/database.go`. Tables are created automatically with `CREATE TABLE IF NOT EXISTS` on startup. The schema includes:

- `organizations`, `teams`, `agents` -- Organizational hierarchy
- `model_pricing` -- LLM model cost reference data
- `budgets` -- Budget limits per entity
- `api_requests` -- TimescaleDB hypertable for request metadata

### Economist Schema

Economist uses SQLAlchemy ORM models defined in `economist/pkg/database.py`. Tables are created with `Base.metadata.create_all()`. The schema includes:

- `cloud_costs` -- Cloud cost line items
- `optimization_recommendations` -- Generated savings recommendations
- `governance_policies` -- Cost governance rules
- `policy_violations` -- Detected policy violations

### Schema Changes

- **Cerebra/Aegis (Go):** Modify the schema SQL in `database.go`. Use `CREATE TABLE IF NOT EXISTS` and `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` for backward-compatible migrations.
- **Economist (Python):** Add or modify SQLAlchemy ORM models in `database.py`. Use `create_tables()` for forward-compatible creation. For production, consider adding Alembic for migration management.

---

## Contributing Workflow

### 1. Find or Create an Issue

Browse [GitHub Issues](https://github.com/bigdegenenergy/open-cloud-ops/issues) for something to work on, or create a new issue describing your proposed change.

### 2. Fork and Branch

```bash
# Fork the repository on GitHub, then:
git clone https://github.com/YOUR_USERNAME/open-cloud-ops.git
cd open-cloud-ops
git remote add upstream https://github.com/bigdegenenergy/open-cloud-ops.git

# Create a feature branch
git checkout -b feature/your-feature-name
```

### 3. Develop and Test

- Make your changes following the code style guidelines
- Write tests for new functionality
- Run the full test suite to ensure nothing is broken
- Update documentation if your change affects API surfaces

### 4. Commit and Push

```bash
# Stage your changes
git add .

# Commit with a conventional commit message
git commit -m "feat(cerebra): add per-model budget limits"

# Push to your fork
git push origin feature/your-feature-name
```

### 5. Open a Pull Request

Open a PR against the `main` branch of the upstream repository. In the PR description:

- Reference the related issue (e.g., `Closes #42`)
- Describe what changes you made and why
- Note any migration or configuration changes required
- Include screenshots for UI changes

### 6. Code Review

- A maintainer will review your PR
- Address any requested changes
- Once approved, a maintainer will merge the PR

### PR Checklist

Before submitting your PR, verify:

- [ ] Code compiles/runs without errors
- [ ] All existing tests pass
- [ ] New tests added for new functionality
- [ ] Code formatted with `gofmt` / `black` / `prettier`
- [ ] Linting passes (`golangci-lint` / `ruff` / `eslint`)
- [ ] Documentation updated if API changed
- [ ] Commit messages follow Conventional Commits
- [ ] No secrets, credentials, or API keys included

---

## Common Development Tasks

### Adding a New API Endpoint to Cerebra

1. Define the route in the HTTP server setup (`cmd/main.go` or a dedicated routes file)
2. Create a handler function in the appropriate `internal/` package
3. Add any required data models to `pkg/models/models.go`
4. Add database schema changes to `pkg/database/database.go` if needed
5. Write unit tests for the handler
6. Update `docs/api-reference/cerebra-api.md`

### Adding a New API Endpoint to Economist

1. Add the route handler in `cmd/main.py` or a dedicated router in `api/`
2. Create the business logic in the appropriate `internal/` package
3. Add any required ORM models to `pkg/database.py`
4. Write tests using `httpx.AsyncClient`
5. Update `docs/api-reference/economist-api.md`
6. Check auto-generated docs at `http://localhost:8081/docs`

### Adding a New Cloud Provider to Economist

1. Create a new adapter file in `economist/pkg/cloud/` (e.g., `oracle.py`)
2. Implement the cost data ingestion interface
3. Add provider credentials to `pkg/config.py`
4. Add the provider to the ingestion orchestrator in `internal/ingestion/`
5. Write tests with mocked API responses
6. Update `.env.example` with the new credential variables

### Adding a New LLM Provider to Cerebra

1. Add the provider constant to `internal/proxy/handler.go`
2. Add the proxy route mapping for the provider's API endpoints
3. Add pricing data to the `model_pricing` table
4. Add token extraction logic for the provider's response format
5. Write integration tests with the provider's API
6. Update `docs/api-reference/cerebra-api.md`

### Debugging Database Queries

```bash
# View recent API requests (Cerebra)
docker exec -it oco-postgres psql -U oco_user -d opencloudops \
  -c "SELECT id, provider, model, cost_usd, timestamp FROM api_requests ORDER BY timestamp DESC LIMIT 10;"

# View cloud costs (Economist)
docker exec -it oco-postgres psql -U oco_user -d opencloudops \
  -c "SELECT provider, service, cost_usd, date FROM cloud_costs ORDER BY date DESC LIMIT 10;"

# View active budgets
docker exec -it oco-postgres psql -U oco_user -d opencloudops \
  -c "SELECT scope, entity_id, limit_usd, spent_usd FROM budgets;"
```

### Resetting the Development Database

```bash
# Drop and recreate the database
docker exec -it oco-postgres psql -U oco_user -d postgres \
  -c "DROP DATABASE IF EXISTS opencloudops; CREATE DATABASE opencloudops;"

# Restart services to reinitialize schema
docker compose -f deploy/docker/docker-compose.dev.yml restart
```
