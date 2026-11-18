
from __future__ import annotations

import hashlib
import re
import threading

from collections import OrderedDict

import numpy as np

from app.schemas.hybrid_search import (
    SearchDocument,
    SearchResult,
)

from app.services.embedding_service import EmbeddingService


class HybridSearchService:
    """
    Hybrid search combining:
    - 80% semantic similarity
    - 20% keyword relevance

    Keyword relevance considers both title and content,
    prioritizing title matches.

    Document embeddings are cached to avoid repeatedly
    processing unchanged documents.
    """

    STOP_WORDS = {
        # English
        "a", "an", "the", "about", "and", "for",
        "from", "in", "of", "on", "to", "with",
        "me", "find", "show", "give", "get",
        "book", "books", "document", "documents",
        "how",

        # Filipino / Tagalog
        "ang", "ng", "mga", "sa", "na", "at",
        "ay", "ito", "iyan", "iyon", "para",
        "po", "ba", "ako", "ko", "mo", "si",
        "ni", "kay", "kina", "kung", "paano",
        "hanapin", "pakihanap", "maghanap",
        "aklat", "libro",
    }

    MAX_CACHE_SIZE = 10_000

    def __init__(
        self,
        embedding_service: EmbeddingService,
    ) -> None:

        self.embedding_service = embedding_service

        # Key: (document_id, content_hash)
        # Value: normalized NumPy embedding
        self._embedding_cache: OrderedDict[
            tuple[int, str],
            np.ndarray
        ] = OrderedDict()

        # Protect cache reads, writes, and embedding generation.
        self._cache_lock = threading.RLock()

    def search(
        self,
        query: str,
        documents: list[SearchDocument],
        top_k: int = 5,
    ) -> list[SearchResult]:

        if not documents:
            return []

        query = query.strip()

        if not query:
            return []

        # Generate one embedding for the search query.
        query_embedding = self.embedding_service.generate(query)

        # Retrieve cached embeddings or generate missing ones.
        document_embeddings = self._get_document_embeddings(
            documents
        )

        # Both embeddings are normalized.
        # Their dot product is cosine similarity.
        semantic_scores = document_embeddings @ query_embedding

        query_keywords = self._extract_keywords(query)

        results: list[SearchResult] = []

        for index, document in enumerate(documents):

            keyword_score = self._keyword_score(
                query_keywords=query_keywords,
                title=document.title,
                content=document.content,
            )

            semantic_score = float(semantic_scores[index])

            # Semantic relevance receives greater weight.
            hybrid_score = (
                (keyword_score * 0.20)
                + (semantic_score * 0.80)
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

        # Sort by relevance, then document ID for deterministic ties.
        results.sort(
            key=lambda result: (
                -result.hybrid_score,
                result.document_id,
            )
        )

        return results[:top_k]

    def _get_document_embeddings(
        self,
        documents: list[SearchDocument],
    ) -> np.ndarray:
        """
        Return embeddings for all documents.

        Existing embeddings are reused.
        Missing or modified documents are encoded together.
        """

        with self._cache_lock:

            document_keys: list[tuple[int, str]] = []
            missing_keys: list[tuple[int, str]] = []
            missing_texts: list[str] = []

            for document in documents:

                document_text = (
                    f"{document.title}. {document.content}"
                )

                content_hash = hashlib.sha256(
                    document_text.encode("utf-8")
                ).hexdigest()

                cache_key = (
                    document.document_id,
                    content_hash,
                )

                document_keys.append(cache_key)

                if cache_key in self._embedding_cache:

                    # Mark recently accessed entries as most recent.
                    self._embedding_cache.move_to_end(cache_key)

                else:

                    missing_keys.append(cache_key)
                    missing_texts.append(document_text)

            # Generate embeddings for uncached documents in one batch.
            if missing_texts:

                new_embeddings = (
                    self.embedding_service.generate_many(
                        missing_texts
                    )
                )

                for key, embedding in zip(
                    missing_keys,
                    new_embeddings,
                ):

                    self._embedding_cache[key] = embedding

                    self._embedding_cache.move_to_end(key)

                # Remove oldest entries when cache exceeds its limit.
                while (
                    len(self._embedding_cache)
                    > self.MAX_CACHE_SIZE
                ):
                    self._embedding_cache.popitem(last=False)

            # Reconstruct embeddings in the original document order.
            return np.stack(
                [
                    self._embedding_cache[key]
                    for key in document_keys
                ]
            )

    def _extract_keywords(
        self,
        query: str,
    ) -> set[str]:

        words = re.findall(
            r"\b[^\W_]+\b",
            query.casefold(),
        )

        return {
            word
            for word in words
            if word not in self.STOP_WORDS
        }

    def _keyword_score(
        self,
        query_keywords: set[str],
        title: str,
        content: str,
    ) -> float:
        """
        Calculate keyword relevance.

        Title matches: 75%
        Content matches: 25%
        """

        if not query_keywords:
            return 0.0

        title_words = set(
            re.findall(
                r"\b[^\W_]+\b",
                title.casefold(),
            )
        )

        content_words = set(
            re.findall(
                r"\b[^\W_]+\b",
                content.casefold(),
            )
        )

        title_matches = query_keywords.intersection(
            title_words
        )

        content_matches = query_keywords.intersection(
            content_words
        )

        title_score = (
            len(title_matches) / len(query_keywords)
        )

        content_score = (
            len(content_matches) / len(query_keywords)
        )

        keyword_score = (
            (title_score * 0.75)
            + (content_score * 0.25)
        )

        return min(keyword_score, 1.0)
