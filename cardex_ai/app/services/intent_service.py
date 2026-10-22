
import re

from app.schemas.search import IntentResponse


class IntentService:

    SEARCH_PATTERNS = {
        "document_search": [
            r"\bbook(s)?\b",
            r"\bdocument(s)?\b",
            r"\bfile(s)?\b",
            r"\bmanual(s)?\b",
            r"\bpaper(s)?\b",
            r"\bpublication(s)?\b",
            r"\b(?:aklat|libro)\b",
            r"\b(?:hanapin|pakihanap|maghanap)\b",
        ],
        "author_search": [
            r"\bauthor\b",
            r"\bwritten by\b",
            r"\bby\s+[a-zA-Z]+\b",
            r"\bmay[- ]akda\b",
            r"\bisinulat ni\b",
            r"\bsinulat ni\b",
        ],
        "topic_search": [
            r"\babout\b",
            r"\bregarding\b",
            r"\brelated to\b",
            r"\bconcerning\b",
            r"\btungkol sa\b",
            r"\bpatungkol sa\b",
            r"\bukol sa\b",
        ],
    }

    STOP_WORDS = {
        # English
        "a", "an", "the", "about", "and", "for",
        "from", "in", "of", "on", "to", "with",
        "me", "find", "show", "give", "get",

        # Filipino / Tagalog
        "ang", "ng", "mga", "sa", "na", "at",
        "ay", "ito", "iyan", "iyon", "para",
        "po", "ba", "ako", "ko", "mo", "si",
        "ni", "kay", "kina", "kung", "paano",
        "hanapin", "pakihanap", "maghanap",
        "aklat", "libro", "tungkol", "patungkol",
    }

    def parse(self, query: str) -> IntentResponse:
        normalized = query.casefold().strip()

        intent_type = self._detect_intent(normalized)
        keywords = self._extract_keywords(normalized)
        topic = self._extract_topic(normalized, keywords)

        return IntentResponse(
            type=intent_type,
            topic=topic,
            keywords=keywords,
        )

    def _detect_intent(self, query: str) -> str:
        # Check more specific intents before general document search.
        for intent in (
            "author_search",
            "topic_search",
            "document_search",
        ):
            for pattern in self.SEARCH_PATTERNS[intent]:
                if re.search(pattern, query):
                    return intent

        return "general_search"

    def _extract_keywords(self, query: str) -> list[str]:
        words = re.findall(r"\b[^\W_]+\b", query)

        keywords = [
            word
            for word in words
            if word not in self.STOP_WORDS
        ]

        return list(dict.fromkeys(keywords))[:10]

    def _extract_topic(
        self,
        query: str,
        keywords: list[str],
    ) -> str:
        if not keywords:
            return query

        return " ".join(keywords)
