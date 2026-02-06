#!/usr/bin/env bash
# =============================================================================
# Open Cloud Ops - Development Environment Setup
# =============================================================================
# Sets up the local development environment by checking prerequisites,
# configuring environment variables, and starting infrastructure services.
#
# Usage:
#   ./scripts/dev-setup.sh
#   make dev-setup
# =============================================================================

set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${PROJECT_ROOT}/deploy/docker/docker-compose.yml"
ENV_FILE="${PROJECT_ROOT}/.env"
ENV_EXAMPLE="${PROJECT_ROOT}/.env.example"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ---------------------------------------------------------------------------
# Helper Functions
# ---------------------------------------------------------------------------
info()  { echo -e "${BLUE}[INFO]${NC}  $*"; }
ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; }

check_command() {
    local cmd="$1"
    local name="${2:-$1}"
    if command -v "$cmd" &> /dev/null; then
        ok "$name is installed ($(command -v "$cmd"))"
        return 0
    else
        error "$name is NOT installed. Please install it before continuing."
        return 1
    fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
echo ""
echo "============================================="
echo "  Open Cloud Ops - Development Setup"
echo "============================================="
echo ""

# Step 1: Check prerequisites
info "Checking prerequisites..."
echo ""

MISSING=0

check_command docker "Docker" || MISSING=1
check_command docker-compose "Docker Compose" || {
    # docker compose (v2 plugin) is also acceptable
    if docker compose version &> /dev/null 2>&1; then
        ok "Docker Compose v2 plugin detected"
    else
        MISSING=1
    fi
}
check_command go "Go" || MISSING=1
check_command python3 "Python 3" || MISSING=1

echo ""

if [ "$MISSING" -ne 0 ]; then
    error "One or more prerequisites are missing. Please install them and re-run this script."
    exit 1
fi

ok "All prerequisites satisfied."
echo ""

# Step 2: Set up .env file
info "Checking environment configuration..."

if [ ! -f "$ENV_FILE" ]; then
    info "Copying .env.example to .env..."
    cp "$ENV_EXAMPLE" "$ENV_FILE"
    ok ".env file created. Please review and update values as needed."
else
    ok ".env file already exists."
fi

echo ""

# Step 3: Start infrastructure services
info "Starting infrastructure services (PostgreSQL, Redis)..."
echo ""

docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d postgres redis

echo ""

# Step 4: Wait for PostgreSQL to be ready
info "Waiting for PostgreSQL to be ready..."

MAX_RETRIES=30
RETRY_INTERVAL=2
RETRIES=0

until docker-compose -f "$COMPOSE_FILE" exec -T postgres pg_isready -U oco_user -d opencloudops &> /dev/null; do
    RETRIES=$((RETRIES + 1))
    if [ "$RETRIES" -ge "$MAX_RETRIES" ]; then
        error "PostgreSQL failed to become ready after $((MAX_RETRIES * RETRY_INTERVAL)) seconds."
        exit 1
    fi
    echo -n "."
    sleep "$RETRY_INTERVAL"
done

echo ""
ok "PostgreSQL is ready."
echo ""

# Step 5: Run init.sql (idempotent - safe to re-run)
info "Running database initialization script..."

docker-compose -f "$COMPOSE_FILE" exec -T postgres \
    psql -U oco_user -d opencloudops -f /docker-entrypoint-initdb.d/init.sql \
    2>&1 | tail -5

ok "Database initialization complete."
echo ""

# Step 6: Print instructions
echo "============================================="
echo "  Development Environment Ready!"
echo "============================================="
echo ""
echo "Infrastructure services running:"
echo "  - PostgreSQL: localhost:5432"
echo "  - Redis:      localhost:6379"
echo ""
echo "To start application services:"
echo ""
echo "  Cerebra (LLM Gateway):"
echo "    cd ${PROJECT_ROOT}/cerebra"
echo "    go run ./cmd/main.go"
echo ""
echo "  Economist (FinOps Core):"
echo "    cd ${PROJECT_ROOT}/economist"
echo "    python3 cmd/main.py"
echo ""
echo "  Aegis (Resilience Engine):"
echo "    cd ${PROJECT_ROOT}/aegis"
echo "    go run ./cmd/main.go"
echo ""
echo "Or start everything with Docker Compose:"
echo "  docker-compose -f ${COMPOSE_FILE} up -d"
echo ""
echo "Useful commands:"
echo "  make test          - Run all tests"
echo "  make lint          - Lint all services"
echo "  make docker-build  - Build all Docker images"
echo "  make docker-down   - Stop all Docker services"
echo ""
