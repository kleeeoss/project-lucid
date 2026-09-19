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

    # AI Provider configuration
    LLM_PRIMARY_PROVIDER: str = "groq"        # groq | gemini | ollama
    LLM_FALLBACK_PROVIDER: str = "gemini"     # groq | gemini | ollama | none
    LLM_TIMEOUT_SECONDS: float = 10.0
    LLM_MAX_TOKENS: int = 1024
    LLM_TEMPERATURE: float = 0.1

    # Provider Models & Keys
    GROQ_API_KEY: str = ""
    GROQ_MODEL: str = "llama-3.1-8b-instant"

    GEMINI_API_KEY: str = ""
    GEMINI_MODEL: str = "gemini-1.5-flash"

    OLLAMA_BASE_URL: str = "http://localhost:11434"
    OLLAMA_MODEL: str = "llama3:8b"


settings = Settings()
