package service

import (
	"errors"

	model "backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
)

func GetPendingProfileChangeRequests(
	db *gorm.DB,
) ([]model.ProfileChangeRequest, error) {

	requests, err := repository.GetPendingProfileChangeRequests(
		db,
	)

	if err != nil {
		return nil, errors.New(
			"failed to retrieve pending profile change requests",
		)
	}

	return requests, nil
}
