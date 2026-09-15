from fastapi import APIRouter

from app.schemas.ranking import (
    RankingRequest,
    RankingResponse,
)
from app.services.ranking_service import RankingService


router = APIRouter(
    prefix="/api/v1/ranking",
    tags=["AI Ranking"],
)

ranking_service = RankingService()


@router.post(
    "/rank",
    response_model=RankingResponse,
)
def rank_results(
    request: RankingRequest,
) -> RankingResponse:

    query = request.query.strip()

    recommendations = ranking_service.rank(
        documents=request.matches,
        user=request.user,
    )

    return RankingResponse(
        query=query,
        recommendations=recommendations,
    )