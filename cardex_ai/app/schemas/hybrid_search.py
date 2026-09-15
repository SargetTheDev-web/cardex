from pydantic import BaseModel, Field


class HybridSearchRequest(BaseModel):
    query: str = Field(
        ...,
        min_length=1,
        max_length=500,
    )

    top_k: int = Field(
        default=5,
        ge=1,
        le=20,
    )


class SearchResult(BaseModel):
    document_id: int
    title: str
    content: str
    keyword_score: float
    semantic_score: float
    hybrid_score: float


class HybridSearchResponse(BaseModel):
    query: str
    matches_found: bool
    results: list[SearchResult]