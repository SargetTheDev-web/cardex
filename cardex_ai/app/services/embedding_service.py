
from sentence_transformers import SentenceTransformer

from app.core.config import get_settings


class EmbeddingService:
    def __init__(self) -> None:
        settings = get_settings()

        self.model = SentenceTransformer(
            settings.embedding_model
        )

    def generate(self, text: str) -> list[float]:
        embedding = self.model.encode(
            text,
            normalize_embeddings=True,
            convert_to_numpy=True,
        )

        return embedding.tolist()

    def generate_batch(
        self,
        texts: list[str],
    ) -> list[list[float]]:
        if not texts:
            return []

        embeddings = self.model.encode(
            texts,
            normalize_embeddings=True,
            convert_to_numpy=True,
            batch_size=32,
            show_progress_bar=False,
        )

        return embeddings.tolist()

    def dimensions(self) -> int:
        return self.model.get_sentence_embedding_dimension()
