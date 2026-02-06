"""
Economist configuration module.

Loads settings from environment variables with sensible defaults.
Uses pydantic-settings for validation and type coercion.
"""

from __future__ import annotations

from functools import lru_cache

from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """Application settings loaded from environment variables."""

    # -------------------------------------------------------------------
    # PostgreSQL
    # -------------------------------------------------------------------
    postgres_host: str = "localhost"
    postgres_port: int = 5432
    postgres_db: str = "opencloudops"
    postgres_user: str = "oco_user"
    postgres_password: str = "change_me"

    # -------------------------------------------------------------------
    # Redis
    # -------------------------------------------------------------------
    redis_host: str = "localhost"
    redis_port: int = 6379

    # -------------------------------------------------------------------
    # Service
    # -------------------------------------------------------------------
    economist_port: int = 8081
    log_level: str = "INFO"

    # -------------------------------------------------------------------
    # AWS
    # -------------------------------------------------------------------
    aws_access_key_id: str = ""
    aws_secret_access_key: str = ""
    aws_region: str = "us-east-1"

    # -------------------------------------------------------------------
    # Azure
    # -------------------------------------------------------------------
    azure_subscription_id: str = ""
    azure_tenant_id: str = ""
    azure_client_id: str = ""
    azure_client_secret: str = ""

    # -------------------------------------------------------------------
    # GCP
    # -------------------------------------------------------------------
    gcp_project_id: str = ""
    gcp_credentials_json: str = ""

    # -------------------------------------------------------------------
    # Derived helpers
    # -------------------------------------------------------------------
    @property
    def postgres_url(self) -> str:
        """Build a full PostgreSQL connection URL."""
        return (
            f"postgresql://{self.postgres_user}:{self.postgres_password}"
            f"@{self.postgres_host}:{self.postgres_port}/{self.postgres_db}"
        )

    @property
    def async_postgres_url(self) -> str:
        """Build an async PostgreSQL connection URL (asyncpg driver)."""
        return (
            f"postgresql+asyncpg://{self.postgres_user}:{self.postgres_password}"
            f"@{self.postgres_host}:{self.postgres_port}/{self.postgres_db}"
        )

    @property
    def redis_url(self) -> str:
        """Build a Redis connection URL."""
        return f"redis://{self.redis_host}:{self.redis_port}/0"

    model_config = {
        "env_prefix": "",
        "env_file": ".env",
        "env_file_encoding": "utf-8",
        "case_sensitive": False,
    }


@lru_cache()
def get_settings() -> Settings:
    """Return a cached singleton of the application settings."""
    return Settings()
