from sentence_transformers import util

from app.repositories.mock_document_repository import Document
from app.schemas.recommendation import RecommendationItem


class RecommendationService:
    """
    Handles semantic document recommendation.

    The service compares the user's query embedding
    against embeddings generated from candidate documents.
    """

    def __init__(self, embedding_service) -> None:

        # Reuse the same embedding model/service from Module 1.
        #
        # This is important because the query and documents
        # need to be represented in the same vector space.
        self.embedding_service = embedding_service

    def recommend(
        self,
        query: str,
        documents: list[Document],
        top_k: int = 5,
    ) -> list[RecommendationItem]:
        """
        Generate recommendations based on semantic similarity.
        """

        # ---------------------------------------------------------
        # STEP 1
        # Generate an embedding for the user's search query.
        # ---------------------------------------------------------

        query_embedding = self.embedding_service.generate(query)

        # ---------------------------------------------------------
        # STEP 2
        # Generate embeddings for all candidate documents.
        #
        # For our prototype, this happens during the request.
        #
        # In the production version, we should generate and store
        # document embeddings when documents are added/updated.
        # ---------------------------------------------------------

        document_embeddings = []

        for document in documents:

            # Combine title and content so the model receives
            # more useful semantic information.
            document_text = (
                f"{document.title}. {document.content}"
            )

            embedding = self.embedding_service.generate(
                document_text
            )

            document_embeddings.append(embedding)

        # ---------------------------------------------------------
        # STEP 3
        # Calculate cosine similarity between the query and
        # every document.
        # ---------------------------------------------------------

        similarities = []

        for document, document_embedding in zip(
            documents,
            document_embeddings,
        ):

            # Convert the vectors into tensors and calculate
            # cosine similarity.
            #
            # Result:
            #     higher score = more semantically similar
            #
            # Example:
            #     0.90 = highly related
            #     0.40 = weakly related
            similarity = util.cos_sim(
                query_embedding,
                document_embedding,
            ).item()

            similarities.append(
                (
                    document,
                    similarity,
                )
            )

        # ---------------------------------------------------------
        # STEP 4
        # Sort documents from highest similarity to lowest.
        # ---------------------------------------------------------

        similarities.sort(
            key=lambda item: item[1],
            reverse=True,
        )

        # ---------------------------------------------------------
        # STEP 5
        # Keep only the requested number of results.
        # ---------------------------------------------------------

        top_results = similarities[:top_k]

        # ---------------------------------------------------------
        # STEP 6
        # Convert internal results into our API response model.
        # ---------------------------------------------------------

        recommendations = []

        for document, similarity in top_results:

            recommendations.append(
                RecommendationItem(
                    document_id=document.document_id,
                    title=document.title,
                    similarity_score=round(
                        similarity,
                        4,
                    ),
                )
            )

        return recommendations