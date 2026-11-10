
from app.services.embedding_service import EmbeddingService
from app.services.hybrid_search_service import HybridSearchService
from app.services.recommendation_service import RecommendationService


# One shared embedding model for the entire application.
embedding_service = EmbeddingService()

hybrid_search_service = HybridSearchService(
    embedding_service=embedding_service,
)

recommendation_service = RecommendationService(
    embedding_service=embedding_service,
)
