from pydantic import BaseModel, Field


class RecommendationRequest(BaseModel):
    # The user's original search query.
    query: str = Field(
        ...,
        min_length=1,
        max_length=500,
        description="Natural-language search query.",
    )

    # Number of recommendations to return.
    # Default is 5, but the client can request fewer/more.
    top_k: int = Field(
        default=5,
        ge=1,
        le=20,
        description="Number of recommendations to return.",
    )


class RecommendationItem(BaseModel):
    # Unique identifier of the document.
    document_id: int

    # Human-readable document title.
    title: str

    # Similarity between the user's query and the document.
    similarity_score: float


class RecommendationResponse(BaseModel):
    # Original query sent by the user.
    query: str

    # Documents ranked by semantic similarity.
    recommendations: list[RecommendationItem]