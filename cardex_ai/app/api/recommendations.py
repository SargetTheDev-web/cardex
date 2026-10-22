from fastapi import APIRouter

from app.repositories.mock_document_repository import (
    MockDocumentRepository,
)
from app.schemas.recommendation import (
    RecommendationRequest,
    RecommendationResponse,
)
from app.services.embedding_service import EmbeddingService
from app.services.recommendation_service import (
    RecommendationService,
)


# ---------------------------------------------------------
# Create a router specifically for recommendation endpoints.
# ---------------------------------------------------------

router = APIRouter(
    prefix="/api/v1/recommendations",
    tags=["Recommendations"],
)


# ---------------------------------------------------------
# Initialize our dependencies.
#
# EmbeddingService loads the Sentence Transformer model.
# RecommendationService uses that model to compare vectors.
# MockDocumentRepository temporarily provides our documents.
# ---------------------------------------------------------

embedding_service = EmbeddingService()

recommendation_service = RecommendationService(
    embedding_service=embedding_service,
)

document_repository = MockDocumentRepository()


@router.post(
    "/generate",
    response_model=RecommendationResponse,
)
def generate_recommendations(
    request: RecommendationRequest,
) -> RecommendationResponse:

    # Remove unnecessary whitespace from the query.
    query = request.query.strip()

    # Get available documents.
    #
    # Currently these come from our mock repository.
    # Later this can come from DMS.
    documents = document_repository.get_all_documents()

    # Run the recommendation algorithm.
    recommendations = recommendation_service.recommend(
        query=query,
        documents=documents,
        top_k=request.top_k,
    )

    # Return the final API response.
    return RecommendationResponse(
        query=query,
        recommendations=recommendations,
    )