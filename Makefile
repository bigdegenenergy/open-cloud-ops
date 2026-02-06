# =============================================================================
# Open Cloud Ops - Makefile
# =============================================================================
# Provides convenient targets for building, testing, linting, and running
# the full Open Cloud Ops platform.
#
# Usage:
#   make help          - Show available targets
#   make dev-setup     - Set up local development environment
#   make build         - Build all services
#   make test          - Run all tests
#   make lint          - Lint all services
#   make docker-build  - Build Docker images
#   make docker-up     - Start all services via Docker Compose
#   make docker-down   - Stop all Docker Compose services
#   make clean         - Remove build artifacts
# =============================================================================

.PHONY: help dev-setup build test lint docker-build docker-up docker-down clean \
        build-cerebra build-economist build-aegis \
        test-cerebra test-economist test-aegis \
        lint-cerebra lint-economist lint-aegis

# Default target
.DEFAULT_GOAL := help

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
PROJECT_ROOT := $(shell pwd)
COMPOSE_FILE := $(PROJECT_ROOT)/deploy/docker/docker-compose.yml
ENV_FILE     := $(PROJECT_ROOT)/.env

# ---------------------------------------------------------------------------
# Help
# ---------------------------------------------------------------------------
help: ## Show this help message
	@echo ""
	@echo "Open Cloud Ops - Available Targets"
	@echo "=================================="
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
	@echo ""

# ---------------------------------------------------------------------------
# Development Setup
# ---------------------------------------------------------------------------
dev-setup: ## Set up the local development environment
	@chmod +x scripts/dev-setup.sh
	@bash scripts/dev-setup.sh

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------
build: build-cerebra build-aegis build-economist ## Build all services

build-cerebra: ## Build Cerebra (Go)
	@echo "Building Cerebra..."
	cd cerebra && CGO_ENABLED=0 go build -o bin/cerebra ./cmd/main.go
	@echo "Cerebra built: cerebra/bin/cerebra"

build-aegis: ## Build Aegis (Go)
	@echo "Building Aegis..."
	cd aegis && CGO_ENABLED=0 go build -o bin/aegis ./cmd/main.go
	@echo "Aegis built: aegis/bin/aegis"

build-economist: ## Build Economist (Python) - install dependencies
	@echo "Building Economist (installing dependencies)..."
	cd economist && pip install -r requirements.txt
	@echo "Economist dependencies installed."

# ---------------------------------------------------------------------------
# Test
# ---------------------------------------------------------------------------
test: ## Run all tests
	@chmod +x scripts/run-tests.sh
	@bash scripts/run-tests.sh

test-cerebra: ## Run Cerebra tests only
	cd cerebra && go test -v -race ./...

test-aegis: ## Run Aegis tests only
	cd aegis && go test -v -race ./...

test-economist: ## Run Economist tests only
	cd economist && python3 -m pytest -v .

# ---------------------------------------------------------------------------
# Lint
# ---------------------------------------------------------------------------
lint: lint-cerebra lint-aegis lint-economist ## Lint all services

lint-cerebra: ## Lint Cerebra (Go)
	@echo "Linting Cerebra..."
	cd cerebra && golangci-lint run --timeout=5m ./...

lint-aegis: ## Lint Aegis (Go)
	@echo "Linting Aegis..."
	cd aegis && golangci-lint run --timeout=5m ./...

lint-economist: ## Lint Economist (Python)
	@echo "Linting Economist..."
	cd economist && ruff check .

# ---------------------------------------------------------------------------
# Docker
# ---------------------------------------------------------------------------
docker-build: ## Build all Docker images
	@echo "Building Docker images..."
	docker-compose -f $(COMPOSE_FILE) build cerebra economist aegis

docker-up: ## Start all services via Docker Compose
	docker-compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) up -d

docker-down: ## Stop all Docker Compose services
	docker-compose -f $(COMPOSE_FILE) down

docker-logs: ## Tail logs from all services
	docker-compose -f $(COMPOSE_FILE) logs -f

docker-ps: ## Show running containers
	docker-compose -f $(COMPOSE_FILE) ps

# ---------------------------------------------------------------------------
# Clean
# ---------------------------------------------------------------------------
clean: ## Remove build artifacts
	@echo "Cleaning build artifacts..."
	rm -f cerebra/bin/cerebra
	rm -f aegis/bin/aegis
	rm -rf cerebra/bin/
	rm -rf aegis/bin/
	rm -rf economist/__pycache__/
	rm -rf economist/.pytest_cache/
	rm -rf economist/.mypy_cache/
	rm -f cerebra/coverage.out
	rm -f aegis/coverage.out
	rm -f economist/coverage.xml
	@echo "Clean complete."
