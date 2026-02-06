# Contributing to Open Cloud Ops

Thank you for your interest in contributing to Open Cloud Ops! This document provides guidelines and information for contributors.

## Getting Started

1. **Fork the repository** on GitHub.
2. **Clone your fork** locally: `git clone https://github.com/YOUR_USERNAME/open-cloud-ops.git`
3. **Create a branch** for your feature or fix: `git checkout -b feature/your-feature-name`
4. **Make your changes** and commit them with clear, descriptive messages.
5. **Push your branch** and open a Pull Request against the `main` branch.

## Development Environment

### Prerequisites

- Docker & Docker Compose
- Go 1.21+ (for Cerebra and Aegis)
- Python 3.11+ (for Economist)
- Node.js 22+ with pnpm (for the Dashboard)
- PostgreSQL 16+ with TimescaleDB
- Redis 7+

### Setting Up

```bash
# Start infrastructure dependencies
docker compose -f deploy/docker/docker-compose.dev.yml up -d

# Install Go dependencies
cd cerebra && go mod download

# Install Python dependencies
cd economist && pip install -r requirements.txt

# Install frontend dependencies
cd cerebra/web/dashboard && pnpm install
```

## Code Style

- **Go:** Follow the standard Go formatting guidelines. Run `gofmt` and `golint` before committing.
- **Python:** Follow PEP 8. Use `black` for formatting and `ruff` for linting.
- **TypeScript/React:** Use Prettier for formatting and ESLint for linting.

## Commit Messages

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

- `feat:` A new feature
- `fix:` A bug fix
- `docs:` Documentation changes
- `style:` Code style changes (formatting, etc.)
- `refactor:` Code refactoring
- `test:` Adding or updating tests
- `chore:` Maintenance tasks

**Example:** `feat(cerebra): add budget enforcement for per-agent limits`

## Pull Request Process

1. Ensure your code passes all existing tests and linting checks.
2. Add tests for any new functionality.
3. Update documentation as needed.
4. Fill out the PR template completely.
5. Request a review from a maintainer.

## Reporting Issues

Use the GitHub Issues tab to report bugs or request features. Please include:

- A clear, descriptive title.
- Steps to reproduce the issue (for bugs).
- Expected vs. actual behavior.
- Your environment details (OS, Go/Python/Node version, etc.).

## Code of Conduct

This project adheres to the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/version/2/1/code_of_conduct/). By participating, you are expected to uphold this code.

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
