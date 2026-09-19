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
    LLM_PRIMARY_PROVIDER: str = "groq"
    LLM_FALLBACK_PROVIDER: str = "gemini"
    LLM_TIMEOUT_SECONDS: float = 10.0
    LLM_MAX_TOKENS: int = 1024
    LLM_TEMPERATURE: float = 0.1

    # Provider Models & Keys
    GROQ_API_KEY: str = ""
    GROQ_MODEL: str = "openai/gpt-oss-20b"

    GEMINI_API_KEY: str = ""
    GEMINI_MODEL: str = "gemini-2.5-flash"

    OLLAMA_BASE_URL: str = "http://localhost:11434"
    OLLAMA_MODEL: str = "llama3:8b"

    # Sandbox Configuration
    SANDBOX_IMAGE_TAG: str = "lucid-sandbox-runner:latest"
    SANDBOX_MEMORY_LIMIT: str = "512m"
    SANDBOX_CPU_LIMIT: float = 1.0
    SANDBOX_PID_LIMIT: int = 100
    SANDBOX_DEFAULT_TIMEOUT: int = 60
    SANDBOX_BASE_TEMP_DIR: str = "/tmp/lucid-sandboxes"


settings = Settings()
