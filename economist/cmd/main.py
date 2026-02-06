"""
Economist - Open Cloud Ops FinOps Core

A multi-cloud cost management and optimization engine that integrates
with AWS, Azure, and GCP to provide cost visibility, optimization
recommendations, and governance frameworks.
"""

from __future__ import annotations

import logging
import os
import sys

import uvicorn
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

# ---------------------------------------------------------------------------
# Ensure the project root is on ``sys.path`` so absolute imports work when
# running ``python cmd/main.py`` from the ``economist/`` directory.
# ---------------------------------------------------------------------------
_PROJECT_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
if _PROJECT_ROOT not in sys.path:
    sys.path.insert(0, _PROJECT_ROOT)

from api.routes import configure_routes, router as api_router  # noqa: E402
from internal.ingestion.collector import CostCollector  # noqa: E402
from internal.optimizer.engine import OptimizationEngine  # noqa: E402
from internal.policy.engine import PolicyEngine  # noqa: E402
from pkg.config import get_settings  # noqa: E402
from pkg.database import create_tables, init_engine  # noqa: E402

logger = logging.getLogger("economist")

# ---------------------------------------------------------------------------
# Application factory
# ---------------------------------------------------------------------------


def create_app() -> FastAPI:
    """Build and configure the FastAPI application."""

    settings = get_settings()

    # -- Logging -------------------------------------------------------
    logging.basicConfig(
        level=getattr(logging, settings.log_level.upper(), logging.INFO),
        format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
    )

    # -- FastAPI --------------------------------------------------------
    app = FastAPI(
        title="Economist - Open Cloud Ops FinOps Core",
        description=(
            "Multi-cloud cost management and optimization engine providing "
            "cost visibility, optimization recommendations, and governance."
        ),
        version="0.1.0",
    )

    # -- CORS -----------------------------------------------------------
    app.add_middleware(
        CORSMiddleware,
        allow_origins=["*"],
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )

    # -- Singleton services --------------------------------------------
    collector = CostCollector()
    optimizer = OptimizationEngine()
    policy_engine = PolicyEngine()

    # Register cloud providers (only when credentials are configured)
    _register_providers(collector, settings)

    # Inject services into the route module
    configure_routes(collector, optimizer, policy_engine)

    # -- Routers -------------------------------------------------------
    app.include_router(api_router)

    # -- Health check (standalone, outside versioned router) ------------
    @app.get("/health")
    async def health_check():
        return {"status": "healthy", "service": "economist"}

    # -- Lifecycle events -----------------------------------------------
    @app.on_event("startup")
    async def on_startup():
        logger.info("Initializing database engine ...")
        try:
            init_engine(settings.postgres_url)
            create_tables()
            logger.info("Database tables created / verified.")
        except Exception as exc:
            logger.warning(
                "Could not connect to database at startup: %s. "
                "The application will start but database-dependent "
                "endpoints may fail.",
                exc,
            )

    @app.on_event("shutdown")
    async def on_shutdown():
        logger.info("Economist shutting down.")

    return app


def _register_providers(collector: CostCollector, settings) -> None:
    """Register cloud provider adapters whose credentials are configured."""
    if settings.aws_access_key_id or settings.aws_region:
        try:
            from pkg.cloud.aws import AWSProvider

            collector.register_provider("aws", AWSProvider(settings))
            logger.info("AWS provider registered.")
        except Exception as exc:
            logger.warning("Could not register AWS provider: %s", exc)

    if settings.azure_subscription_id:
        try:
            from pkg.cloud.azure import AzureProvider

            collector.register_provider("azure", AzureProvider(settings))
            logger.info("Azure provider registered.")
        except Exception as exc:
            logger.warning("Could not register Azure provider: %s", exc)

    if settings.gcp_project_id:
        try:
            from pkg.cloud.gcp import GCPProvider

            collector.register_provider("gcp", GCPProvider(settings))
            logger.info("GCP provider registered.")
        except Exception as exc:
            logger.warning("Could not register GCP provider: %s", exc)


# ---------------------------------------------------------------------------
# Module-level app instance (used by ``uvicorn cmd.main:app``)
# ---------------------------------------------------------------------------
app = create_app()

if __name__ == "__main__":
    settings = get_settings()
    port = settings.economist_port
    print("==============================================")
    print("  Economist - Open Cloud Ops FinOps Core")
    print("==============================================")
    uvicorn.run(app, host="0.0.0.0", port=port)
