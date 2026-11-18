// internal/service/user_report_service.go

package service

import (
	"backend/internal/repository"

	"gorm.io/gorm"
)

func GetUserReport(
	db *gorm.DB,
) ([]repository.UserReportRow, error) {

	return repository.GetUserReport(db)
}
