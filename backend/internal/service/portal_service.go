// internal/service/portal_service.go

package service

import (
	"backend/internal/client"
	"backend/internal/repository"

	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

type PortalDashboard struct {
	Statistics             repository.PortalStatistics    `json:"statistics"`
	RecentTransactions     []repository.PortalTransaction `json:"recent_transactions"`
	Recommendations        []client.Recommendation        `json:"recommendations"`
	RecommendationStrategy string                         `json:"recommendation_strategy"`
}

type PortalService struct {
	db       *gorm.DB
	aiClient *client.AIClient
}

func NewPortalService(db *gorm.DB) *PortalService {
	return &PortalService{
		db:       db,
		aiClient: client.NewAIClientFromEnv(),
	}
}

func (s *PortalService) GetDashboard(
	userID int,
) (PortalDashboard, error) {
	statistics, err := repository.GetPortalStatistics(
		s.db,
		userID,
	)
	if err != nil {
		return PortalDashboard{}, err
	}

	transactions, err := repository.GetPortalRecentTransactions(
		s.db,
		userID,
		10,
	)
	if err != nil {
		return PortalDashboard{}, err
	}

	dashboard := PortalDashboard{
		Statistics:             statistics,
		RecentTransactions:     transactions,
		Recommendations:        []client.Recommendation{},
		RecommendationStrategy: "UNAVAILABLE",
	}

	// AI recommendations are optional.
	// A Python service failure must not take down the portal.
	if s.aiClient == nil || !s.aiClient.Enabled() {
		return dashboard, nil
	}

	history, err := repository.GetPortalBorrowingHistory(
		s.db,
		userID,
		20,
	)
	if err != nil {
		return PortalDashboard{}, err
	}

	candidates, err := repository.GetPortalRecommendationCandidates(
		s.db,
		userID,
		200,
	)
	if err != nil {
		return PortalDashboard{}, err
	}

	if len(history) == 0 {
		dashboard.RecommendationStrategy = "INSUFFICIENT_HISTORY"
		return dashboard, nil
	}

	if len(candidates) == 0 {
		dashboard.RecommendationStrategy = "NO_CANDIDATES"
		return dashboard, nil
	}

	borrowedBooks := make(
		[]client.BookMetadata,
		0,
		len(history),
	)

	for _, book := range history {
		borrowedBooks = append(
			borrowedBooks,
			client.BookMetadata{
				BookID:              book.BookID,
				Title:               book.Title,
				Description:         book.Description,
				Category:            book.Category,
				CategoryDescription: book.CategoryDescription,
				Authors:             book.Authors,
			},
		)
	}

	candidateBooks := make(
		[]client.BookMetadata,
		0,
		len(candidates),
	)

	for _, book := range candidates {
		candidateBooks = append(
			candidateBooks,
			client.BookMetadata{
				BookID:              book.BookID,
				Title:               book.Title,
				Description:         book.Description,
				Category:            book.Category,
				CategoryDescription: book.CategoryDescription,
				Authors:             book.Authors,
			},
		)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		35*time.Second,
	)
	defer cancel()

	aiResult, err := s.aiClient.Recommend(
		ctx,
		client.RecommendationRequest{
			BorrowedBooks: borrowedBooks,
			Candidates:    candidateBooks,
			TopK:          5,
		},
	)

	if err != nil {
		slog.Error(
			"Portal recommendation request failed",
			"error", err,
			"user_id", userID,
		)

		dashboard.RecommendationStrategy = "AI_UNAVAILABLE"
		return dashboard, nil
	}

	dashboard.Recommendations = aiResult.Recommendations
	dashboard.RecommendationStrategy = aiResult.Strategy

	return dashboard, nil
}
