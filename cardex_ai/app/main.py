from fastapi import FastAPI

from app.core.config import get_settings

from app.api.hybrid_search import router as hybrid_search_router



settings = get_settings()

app = FastAPI(
    title=settings.app_name,
    version="1.0.0",
)

# AI Module 1: Search analysis, intent recognition, and hybrid search
app.include_router(hybrid_search_router)

# AI Module 2: Semantic recommendations

# AI Module 3: Machine learning ranking


@app.get("/health", tags=["Health"])
def health_check() -> dict[str, str]:
    return {
        "status": "healthy",
        "service": "cardex-ai",
    }
