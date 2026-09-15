from fastapi import APIRouter, HTTPException

from app.repositories.mock_document_repository import (
    MockDocumentRepository,
)
from app.schemas.hybrid_search import (
    HybridSearchRequest,
    HybridSearchResponse,
)
from app.services.embedding_service import EmbeddingService
from app.services.hybrid_search_service import (
    HybridSearchService,
)


router = APIRouter(
    prefix="/api/v1/search",
    tags=["Hybrid Search"],
)

embedding_service = EmbeddingService()

hybrid_search_service = HybridSearchService(
    embedding_service=embedding_service,
)

document_repository = MockDocumentRepository()


@router.post(
    "/hybrid",
    response_model=HybridSearchResponse,
)
def hybrid_search(
    request: HybridSearchRequest,
) -> HybridSearchResponse:

    query = request.query.strip()

    if not query:
        raise HTTPException(
            status_code=400,
            detail="Search query cannot be empty.",
        )

    documents = document_repository.get_all_documents()

    results = hybrid_search_service.search(
        query=query,
        documents=documents,
        top_k=request.top_k,
    )

    # For now, a result with a meaningful hybrid score
    # is considered a match.
    matches_found = any(
        result.hybrid_score > 0.25
        for result in results
    )

    return HybridSearchResponse(
        query=query,
        matches_found=matches_found,
        results=results,
    )