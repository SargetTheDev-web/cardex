from sklearn.linear_model import LogisticRegression

from app.schemas.ranking import (
    MatchingDocument,
    RankedRecommendation,
    UserProfile,
)


class RankingService:
    """
    Personalizes and reranks documents that were
    already found by Hybrid Search.
    """

    def __init__(self) -> None:
        self.model = self._train_model()

    def _train_model(self) -> LogisticRegression:
        """
        Temporary training data.

        This will eventually be replaced with real
        user interaction data from CARDex.
        """

        training_features = [
            [0.9, 1.0, 1.0, 1.0],
            [0.8, 1.0, 0.5, 1.0],
            [0.7, 0.5, 1.0, 0.0],
            [0.4, 0.0, 0.0, 0.0],
            [0.3, 0.0, 0.5, 0.0],
            [0.2, 0.0, 0.0, 0.0],
        ]

        training_labels = [
            1,
            1,
            1,
            0,
            0,
            0,
        ]

        model = LogisticRegression()

        model.fit(
            training_features,
            training_labels,
        )

        return model

    def rank(
        self,
        documents: list[MatchingDocument],
        user: UserProfile,
    ) -> list[RankedRecommendation]:

        ranked_results = []

        for document in documents:

            topic_score = self._topic_match(
                document,
                user,
            )

            keyword_score = self._keyword_match(
                document,
                user,
            )

            history_score = 0.0

            features = [[
                document.hybrid_score,
                topic_score,
                keyword_score,
                history_score,
            ]]

            personalization_score = (
                self.model.predict_proba(features)[0][1]
            )

            final_score = (
                (document.hybrid_score * 0.60)
                + (personalization_score * 0.40)
            )

            ranked_results.append(
                RankedRecommendation(
                    document_id=document.document_id,
                    title=document.title,
                    hybrid_score=round(
                        document.hybrid_score,
                        4,
                    ),
                    personalization_score=round(
                        personalization_score,
                        4,
                    ),
                    final_score=round(
                        final_score,
                        4,
                    ),
                )
            )

        ranked_results.sort(
            key=lambda result: result.final_score,
            reverse=True,
        )

        return ranked_results

    def _topic_match(
        self,
        document: MatchingDocument,
        user: UserProfile,
    ) -> float:

        if not user.preferred_topics:
            return 0.0

        text = (
            f"{document.title} "
            f"{document.content}"
        ).lower()

        matches = sum(
            1
            for topic in user.preferred_topics
            if topic.lower() in text
        )

        return min(
            matches / len(user.preferred_topics),
            1.0,
        )

    def _keyword_match(
        self,
        document: MatchingDocument,
        user: UserProfile,
    ) -> float:

        if not user.preferred_keywords:
            return 0.0

        text = (
            f"{document.title} "
            f"{document.content}"
        ).lower()

        matches = sum(
            1
            for keyword in user.preferred_keywords
            if keyword.lower() in text
        )

        return min(
            matches / len(user.preferred_keywords),
            1.0,
        )