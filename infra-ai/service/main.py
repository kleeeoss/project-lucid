"""
Main FastAPI application entry point.
"""

from contextlib import asynccontextmanager
from fastapi import FastAPI, Request, status
from fastapi.exceptions import RequestValidationError
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
import structlog

from service.config import settings
from service.routes import router

logger = structlog.get_logger()


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup hook
    logger.info("service_startup", service=settings.SERVICE_NAME, environment=settings.ENVIRONMENT)
    yield
    # Shutdown hook
    logger.info("service_shutdown", service=settings.SERVICE_NAME)


app = FastAPI(
    title="Lucid-CI — AI Remediation & Sandbox Service",
    description="Microservice providing SLM remediation and dynamic sandbox detonation.",
    version="1.0.0",
    lifespan=lifespan,
)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Restricted via Caddy reverse proxy in cloud
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.exception_handler(RequestValidationError)
async def validation_exception_handler(request: Request, exc: RequestValidationError):
    """Format Pydantic schema validation failures with clean JSON output."""
    logger.warning("request_validation_failed", path=request.url.path, errors=exc.errors())
    return JSONResponse(
        status_code=status.HTTP_422_UNPROCESSABLE_ENTITY,
        content={
            "error": "SCHEMA_VALIDATION_ERROR",
            "message": "The request payload failed contract validation.",
            "details": exc.errors(),
        },
    )


# Attach API router
app.include_router(router)


if __name__ == "__main__":
    import uvicorn
    uvicorn.run("service.main:app", host=settings.HOST, port=settings.PORT, reload=True)