
from pydantic import BaseModel, Field


class SearchDocument(BaseModel):
    document_id: int
    title: str = Field(..., min_length=1, max_length=500)
    content: str = Field(default="", max_length=10000)


class HybridSearchRequest(BaseModel):
    query: str = Field(..., min_length=1, max_length=500)

    top_k: int = Field(default=10, ge=1, le=10)

    documents: list[SearchDocument] = Field(
        ...,
        min_length=1,
        max_length=1000,
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
