from pydantic import BaseModel, Field


class SearchAnalyzeRequest(BaseModel):
    query: str = Field(
        ...,
        min_length=1,
        max_length=500,
        description="Natural-language search query.",
    )


class IntentResponse(BaseModel):
    type: str
    topic: str
    keywords: list[str]


class SearchAnalyzeResponse(BaseModel):
    query: str
    intent: IntentResponse
    embedding: list[float]
    embedding_dimensions: int