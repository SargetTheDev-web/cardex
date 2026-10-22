
import re

from sentence_transformers import util

from app.schemas.hybrid_search import (
    SearchDocument,
    SearchResult,
)


class HybridSearchService:
    """
    Hybrid search using keyword matching and
    multilingual semantic similarity.
    """

    STOP_WORDS = {
        # English
        "a", "an", "the", "about", "and", "for",
        "from", "in", "of", "on", "to", "with",
        "me", "find", "show", "give", "get",
        "book", "books", "document", "documents",

        # Filipino / Tagalog
        "ang", "ng", "mga", "sa", "na", "at",
        "ay", "ito", "iyan", "iyon", "para",
        "po", "ba", "ako", "ko", "mo", "si",
        "ni", "kay", "kina", "kung", "paano",
        "hanapin", "pakihanap", "maghanap",
        "aklat", "libro",
    }

    def __init__(self, embedding_service) -> None:
        self.embedding_service = embedding_service

    def search(
        self,
        query: str,
        documents: list[SearchDocument],
        top_k: int = 5,
    ) -> list[SearchResult]:

        query_embedding = self.embedding_service.generate(query)
        query_keywords = self._extract_keywords(query)
        results = []

        for document in documents:
            document_text = f"{document.title}. {document.content}"

            keyword_score = self._keyword_score(
                query_keywords,
                document_text,
            )

            document_embedding = self.embedding_service.generate(
                document_text
            )

            semantic_score = util.cos_sim(
                query_embedding,
                document_embedding,
            ).item()

            hybrid_score = (
                (keyword_score * 0.40)
                + (semantic_score * 0.60)
            )

            results.append(
                SearchResult(
                    document_id=document.document_id,
                    title=document.title,
                    content=document.content,
                    keyword_score=round(keyword_score, 4),
                    semantic_score=round(semantic_score, 4),
                    hybrid_score=round(hybrid_score, 4),
                )
            )

        results.sort(
            key=lambda result: result.hybrid_score,
            reverse=True,
        )

        return results[:top_k]

    def _extract_keywords(self, query: str) -> set[str]:
        words = re.findall(r"\b[^\W_]+\b", query.casefold())

        return {
            word
            for word in words
            if word not in self.STOP_WORDS
        }

    def _keyword_score(
        self,
        query_keywords: set[str],
        document_text: str,
    ) -> float:

        if not query_keywords:
            return 0.0

        document_words = set(
            re.findall(
                r"\b[^\W_]+\b",
                document_text.casefold(),
            )
        )

        matches = query_keywords.intersection(document_words)

        return len(matches) / len(query_keywords)
