"""
Service runtime configuration loaded from environment variables.
"""

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )

    SERVICE_NAME: str = "lucid-ai"
    ENVIRONMENT: str = "development"  # development | staging | production
    PORT: int = 8000
    HOST: str = "0.0.0.0"
    LOG_LEVEL: str = "info"

    # AI Provider configuration (Phase 2)
    LLM_PRIMARY_PROVIDER: str = "groq"
    LLM_FALLBACK_PROVIDER: str = "gemini"
    GROQ_API_KEY: str = ""
    GEMINI_API_KEY: str = ""


settings = Settings()