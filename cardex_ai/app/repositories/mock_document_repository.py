from dataclasses import dataclass


@dataclass
class Document:
    """
    Represents the minimum information the recommendation
    engine needs from a document.
    """

    document_id: int
    title: str
    content: str


class MockDocumentRepository:
    """
    Temporary document repository.

    This simulates documents coming from the DMS.

    Later, this repository can be replaced with a DMS
    repository/API client without changing the recommendation
    algorithm.
    """

    def __init__(self) -> None:

        # Temporary documents for development and testing.
        self.documents = [
            Document(
                document_id=1,
                title="Mountain Racing Techniques",
                content=(
                    "A guide to racing motorcycles through "
                    "mountain roads, corners, elevation changes, "
                    "and challenging mountain passes."
                ),
            ),
            Document(
                document_id=2,
                title="Advanced Motorcycle Cornering",
                content=(
                    "Techniques for motorcycle cornering, braking, "
                    "acceleration, and maintaining control on roads."
                ),
            ),
            Document(
                document_id=3,
                title="Wireless Network Security",
                content=(
                    "Fundamentals of protecting wireless networks "
                    "against unauthorized access and attacks."
                ),
            ),
            Document(
                document_id=4,
                title="Database Management Systems",
                content=(
                    "Introduction to relational databases, SQL, "
                    "database design, normalization, and transactions."
                ),
            ),
            Document(
                document_id=5,
                title="Road Safety and Defensive Riding",
                content=(
                    "Principles of defensive motorcycle riding, "
                    "hazard awareness, traffic safety, and braking."
                ),
            ),

            Document(
                document_id=6,
                title="How to improve your chess openings",
                content=(
                    "Proper way on how to make your openings, "
                    "Theories to learn and master for your chess openings."
                ),
            ),
        ]

    def get_all_documents(self) -> list[Document]:
        """
        Returns all available documents.

        Later, this method can retrieve documents from DMS
        instead of this hard-coded list.
        """

        return self.documents