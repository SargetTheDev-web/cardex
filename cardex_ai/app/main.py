from fastapi import FastAPI

from app.api.search import router as search_router
from app.api.recommendations import (
    router as recommendation_router,
)
from app.core.config import get_settings


# Load application configuration.
settings = get_settings()


# Create the FastAPI application.
app = FastAPI(
    title=settings.app_name,
    version="1.0.0",
)


# ---------------------------------------------------------
# Register API routers.
# ---------------------------------------------------------

# Module 1:
# Search analysis + intent + embeddings
app.include_router(search_router)


# Module 2:
# Semantic recommendations
app.include_router(recommendation_router)


@app.get("/health")
def health_check() -> dict[str, str]:
    """
    Basic health-check endpoint.
    """

    return {
        "status": "healthy",
        "service": "cardex-ai",
    }