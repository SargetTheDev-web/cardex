from pydantic import BaseModel, Field


class UserProfile(BaseModel):
    user_id: int

    preferred_topics: list[str] = Field(
        default_factory=list,
        max_length=20,
    )

    preferred_keywords: list[str] = Field(
        default_factory=list,
        max_length=50,
    )


class MatchingDocument(BaseModel):
    document_id: int
    title: str
    content: str

    keyword_score: float = Field(
        ge=0.0,
        le=1.0,
    )

    semantic_score: float = Field(
        ge=0.0,
        le=1.0,
    )

    hybrid_score: float = Field(
        ge=0.0,
        le=1.0,
    )


class RankingRequest(BaseModel):
    query: str = Field(
        ...,
        min_length=1,
        max_length=500,
    )

    user: UserProfile

    matches: list[MatchingDocument] = Field(
        ...,
        min_length=1,
        max_length=50,
    )


class RankedRecommendation(BaseModel):
    document_id: int
    title: str

    hybrid_score: float
    personalization_score: float
    final_score: float


class RankingResponse(BaseModel):
    query: str
    recommendations: list[RankedRecommendation]