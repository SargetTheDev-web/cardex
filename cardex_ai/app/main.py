from fastapi import FastAPI

from app.api.search import router as search_router
from app.core.config import get_settings


settings = get_settings()

app = FastAPI(
    title=settings.app_name,
    version="1.0.0",
)

app.include_router(search_router)


@app.get("/health")
def health_check() -> dict[str, str]:
    return {
        "status": "healthy",
        "service": "cardex-ai",
    }