// internal/service/book_search_service.go

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	model "backend/internal/models"

	"gorm.io/gorm"
)

const maxAISearchDocuments = 1000

type aiSearchDocument struct {
	DocumentID int    `json:"document_id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
}

type aiHybridSearchRequest struct {
	Query     string             `json:"query"`
	TopK      int                `json:"top_k"`
	Documents []aiSearchDocument `json:"documents"`
}

type aiSearchResult struct {
	DocumentID    int     `json:"document_id"`
	Title         string  `json:"title"`
	Content       string  `json:"content"`
	KeywordScore  float64 `json:"keyword_score"`
	SemanticScore float64 `json:"semantic_score"`
	HybridScore   float64 `json:"hybrid_score"`
}

type aiHybridSearchResponse struct {
	Query        string           `json:"query"`
	MatchesFound bool             `json:"matches_found"`
	Results      []aiSearchResult `json:"results"`
}

// RankedBook contains the original CARDex book record and the
// scores produced by the Python AI service. Book fields are
// flattened into the JSON response.
type RankedBook struct {
	model.Book
	KeywordScore  float64 `json:"keyword_score"`
	SemanticScore float64 `json:"semantic_score"`
	HybridScore   float64 `json:"hybrid_score"`
}

type PublicBookSearchResponse struct {
	Query        string       `json:"query"`
	MatchesFound bool         `json:"matches_found"`
	Results      []RankedBook `json:"results"`
}

func SearchPublicBooks(
	ctx context.Context,
	db *gorm.DB,
	query string,
	topK int,
) (*PublicBookSearchResponse, error) {
	query = strings.TrimSpace(query)

	if query == "" {
		return nil, fmt.Errorf("search query is required")
	}
	if topK < 1 || topK > 10 {
		return nil, fmt.Errorf("top_k must be between 1 and 10")
	}

	books, err := GetPublicBooks(db)
	if err != nil {
		return nil, fmt.Errorf("load public books: %w", err)
	}

	if len(books) == 0 {
		return &PublicBookSearchResponse{
			Query:        query,
			MatchesFound: false,
			Results:      []RankedBook{},
		}, nil
	}

	// The current Python API accepts at most 1,000 documents.
	// Do not silently omit books if the catalog exceeds that limit.
	if len(books) > maxAISearchDocuments {
		return nil, fmt.Errorf(
			"catalog has %d books; AI search currently supports at most %d candidates",
			len(books),
			maxAISearchDocuments,
		)
	}

	documents := make([]aiSearchDocument, 0, len(books))
	booksByID := make(map[int]model.Book, len(books))

	for _, book := range books {
		booksByID[book.BookID] = book

		contentParts := []string{}

		if book.Description != nil && strings.TrimSpace(*book.Description) != "" {
			contentParts = append(contentParts, strings.TrimSpace(*book.Description))
		}
		if book.ISBN != nil && strings.TrimSpace(*book.ISBN) != "" {
			contentParts = append(contentParts, "ISBN: "+strings.TrimSpace(*book.ISBN))
		}
		if book.Edition != nil && strings.TrimSpace(*book.Edition) != "" {
			contentParts = append(contentParts, "Edition: "+strings.TrimSpace(*book.Edition))
		}
		if book.PublicationYear != nil {
			contentParts = append(contentParts, "Publication year: "+strconv.Itoa(*book.PublicationYear))
		}
		if strings.TrimSpace(book.Language) != "" {
			contentParts = append(contentParts, "Language: "+strings.TrimSpace(book.Language))
		}

		documents = append(documents, aiSearchDocument{
			DocumentID: book.BookID,
			Title:      book.Title,
			Content:    strings.Join(contentParts, ". "),
		})
	}

	payload := aiHybridSearchRequest{
		Query:     query,
		TopK:      topK,
		Documents: documents,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode AI search request: %w", err)
	}

	baseURL := strings.TrimRight(
		strings.TrimSpace(os.Getenv("AI_SERVICE_URL")),
		"/",
	)
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		baseURL+"/api/v1/search/hybrid",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("create AI search request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 90 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("AI service request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil, fmt.Errorf("read AI service response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("AI service returned HTTP %d", resp.StatusCode)
	}

	var aiResponse aiHybridSearchResponse
	if err := json.Unmarshal(responseBody, &aiResponse); err != nil {
		return nil, fmt.Errorf("decode AI service response: %w", err)
	}

	rankedBooks := make([]RankedBook, 0, len(aiResponse.Results))
	seen := make(map[int]struct{}, len(aiResponse.Results))

	for _, result := range aiResponse.Results {
		// Trust the AI's ranking order, but only return records that
		// actually exist in the catalog loaded from PostgreSQL.
		book, exists := booksByID[result.DocumentID]
		if !exists {
			continue
		}
		if _, duplicate := seen[result.DocumentID]; duplicate {
			continue
		}
		seen[result.DocumentID] = struct{}{}

		rankedBooks = append(rankedBooks, RankedBook{
			Book:          book,
			KeywordScore:  result.KeywordScore,
			SemanticScore: result.SemanticScore,
			HybridScore:   result.HybridScore,
		})
	}

	// If the AI reports that no candidate passed its match threshold,
	// return an empty result list rather than unrelated books.
	if !aiResponse.MatchesFound {
		rankedBooks = []RankedBook{}
	}

	return &PublicBookSearchResponse{
		Query:        query,
		MatchesFound: aiResponse.MatchesFound && len(rankedBooks) > 0,
		Results:      rankedBooks,
	}, nil
}

// Keep sort imported for any future tie-breaking rules? Not needed here.
var _ = sort.Slice
