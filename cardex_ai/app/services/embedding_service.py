
from __future__ import annotations

import numpy as np
from sentence_transformers import SentenceTransformer

from app.core.config import get_settings


class EmbeddingService:
    """
    Loads the sentence-transformer model once and supports
    both single-text and batch embedding generation.
    """

    def __init__(self) -> None:
        settings = get_settings()

        self.model = SentenceTransformer(
            settings.embedding_model
        )

    def generate(self, text: str) -> np.ndarray:
        """
        Generate one normalized embedding.
        """

        embeddings = self.generate_many([text])
        return embeddings[0]

    def generate_many(
        self,
        texts: list[str],
    ) -> np.ndarray:
        """
        Generate normalized embeddings in batches.

        Returns:
            NumPy array with shape:
            (number_of_texts, embedding_dimensions)
        """

        if not texts:
            return np.empty((0, self.dimensions()), dtype=np.float32)

        embeddings = self.model.encode(
            texts,
            batch_size=64,
            show_progress_bar=False,
            convert_to_numpy=True,
            normalize_embeddings=True,
        )

        return np.asarray(embeddings, dtype=np.float32)

    def dimensions(self) -> int:
        return self.model.get_sentence_embedding_dimension()
