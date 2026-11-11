from fastapi import APIRouter, HTTPException

from app.schemas.search import (
    SearchAnalyzeRequest,
    SearchAnalyzeResponse,
)
from app.services.embedding_service import EmbeddingService
from app.services.intent_service import IntentService


router = APIRouter(
    prefix="/api/v1/search",
    tags=["Search AI"],
)

embedding_service = EmbeddingService()
intent_service = IntentService()


@router.post(
    "/analyze",
    response_model=SearchAnalyzeResponse,
)
def analyze_search(
    request: SearchAnalyzeRequest,
) -> SearchAnalyzeResponse:

    query = request.query.strip()

    if not query:
        raise HTTPException(
            status_code=400,
            detail="Search query cannot be empty.",
        )

    embedding = embedding_service.generate(query)
    intent = intent_service.parse(query)

    return SearchAnalyzeResponse(
        query=query,
        intent=intent,
        embedding=embedding,
        embedding_dimensions=len(embedding),
    )