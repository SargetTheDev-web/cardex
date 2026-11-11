
import secrets

from fastapi import APIRouter, Header, HTTPException, status

from app.core.config import get_settings
from app.schemas.recommendation import (
    RecommendationResponse,
)
from app.services.embedding_service import EmbeddingService
from app.services.recommendation_service import (
    RecommendationService,
)


router = APIRouter(
    prefix="/api/v1/recommendations",
    tags=["Recommendations"],
)

settings = get_settings()

embedding_service = EmbeddingService()

recommendation_service = RecommendationService(
    embedding_service=embedding_service,
)


@router.post(
    "/portal",
    response_model=RecommendationResponse,
)
def generate_portal_recommendations(
    request: PortalRecommendationRequest,
    x_cardex_ai_key: str | None = Header(
        default=None,
        alias="X-CARDex-AI-Key",
    ),
) -> RecommendationResponse:

    configured_key = settings.internal_api_key

    if not configured_key:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Internal AI authentication is not configured.",
        )

    if (
        not x_cardex_ai_key
        or not secrets.compare_digest(
            x_cardex_ai_key,
            configured_key,
        )
    ):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid internal service credentials.",
        )

    borrowed_ids = {
        book.book_id
        for book in request.borrowed_books
    }

    candidate_ids = [
        book.book_id
        for book in request.candidates
    ]

    if len(candidate_ids) != len(set(candidate_ids)):
        raise HTTPException(
            status_code=status.HTTP_422_UNPROCESSABLE_ENTITY,
            detail="Candidate book IDs must be unique.",
        )

    if borrowed_ids.intersection(candidate_ids):
        raise HTTPException(
            status_code=status.HTTP_422_UNPROCESSABLE_ENTITY,
            detail="Borrowed books cannot be recommendation candidates.",
        )

    if not request.borrowed_books or not request.candidates:
        return RecommendationResponse(
            recommendations=[],
            strategy="INSUFFICIENT_HISTORY",
        )

    recommendations = recommendation_service.recommend(
        borrowed_books=request.borrowed_books,
        candidates=request.candidates,
        top_k=request.top_k,
    )

    return RecommendationResponse(
        recommendations=recommendations,
        strategy="CONTENT_BASED",
    )
