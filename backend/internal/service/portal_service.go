// internal/service/portal_service.go

package service

import (
	"backend/internal/repository"

	"gorm.io/gorm"
)

type PortalDashboard struct {
	Statistics         repository.PortalStatistics    `json:"statistics"`
	RecentTransactions []repository.PortalTransaction `json:"recent_transactions"`
}

type PortalService struct {
	db *gorm.DB
}

func NewPortalService(db *gorm.DB) *PortalService {
	return &PortalService{
		db: db,
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

	return PortalDashboard{
		Statistics:         statistics,
		RecentTransactions: transactions,
	}, nil
}
