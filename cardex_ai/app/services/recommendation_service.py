
import numpy as np

from sentence_transformers import util

from app.schemas.recommendation import (
    BookMetadata,
    RecommendationItem,
)


class RecommendationService:
    def __init__(self, embedding_service) -> None:
        self.embedding_service = embedding_service

    @staticmethod
    def _book_text(book: BookMetadata) -> str:
        parts = [
            book.title,
            book.description,
            book.category,
            book.category_description,
            " ".join(book.authors),
        ]

        return ". ".join(
            part.strip()
            for part in parts
            if part and part.strip()
        )

    def recommend(
        self,
        borrowed_books: list[BookMetadata],
        candidates: list[BookMetadata],
        top_k: int = 5,
    ) -> list[RecommendationItem]:

        if not borrowed_books or not candidates:
            return []

        # Build the user's interest profile.
        history_texts = [
            self._book_text(book)
            for book in borrowed_books
        ]

        candidate_texts = [
            self._book_text(book)
            for book in candidates
        ]

        history_embeddings = np.asarray(
            self.embedding_service.generate_batch(
                history_texts
            ),
            dtype=np.float32,
        )

        candidate_embeddings = self.embedding_service.generate_batch(
            candidate_texts
        )

        candidate_embeddings = np.asarray(
            candidate_embeddings,
            dtype=np.float32,
        )

        # More recent books receive greater weight.
        weights = np.asarray(
            [
                1.0 / (1.0 + (index * 0.15))
                for index in range(len(history_embeddings))
            ],
            dtype=np.float32,
        )

        weights /= weights.sum()

        user_profile = np.average(
            history_embeddings,
            axis=0,
            weights=weights,
        )

        # Normalize the combined profile.
        norm = np.linalg.norm(user_profile)

        if norm == 0:
            return []

        user_profile = user_profile / norm

        # Calculate cosine similarity.
        similarities = util.cos_sim(
            user_profile.tolist(),
            candidate_embeddings.tolist(),
        )[0].tolist()

        ranked = sorted(
            zip(candidates, similarities),
            key=lambda item: item[1],
            reverse=True,
        )

        return [
            RecommendationItem(
                book_id=book.book_id,
                title=book.title,
                similarity_score=round(float(score), 4),
            )
            for book, score in ranked[:top_k]
        ]
