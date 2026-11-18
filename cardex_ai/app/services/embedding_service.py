
from __future__ import annotations

import threading

import numpy as np
from sentence_transformers import SentenceTransformer

from app.core.config import get_settings


class EmbeddingService:
    def __init__(self) -> None:
        settings = get_settings()

        self.model = SentenceTransformer(
            settings.embedding_model,
            device="cpu",
        )

        self._inference_lock = threading.Lock()

    def generate(self, text: str) -> np.ndarray:
        embeddings = self.generate_many([text])
        return embeddings[0]

    def generate_many(self, texts: list[str]) -> np.ndarray:
        if not texts:
            return np.empty(
                (0, self.dimensions()),
                dtype=np.float32,
            )

        with self._inference_lock:
            embeddings = self.model.encode(
                texts,
                batch_size=8,
                show_progress_bar=False,
                convert_to_numpy=True,
                normalize_embeddings=True,
            )

        return np.asarray(embeddings, dtype=np.float32)

    def dimensions(self) -> int:
        return self.model.get_sentence_embedding_dimension()
