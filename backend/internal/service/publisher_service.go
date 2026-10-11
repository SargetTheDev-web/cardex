// internal/service/publisher_service.go

package service

import (
	"errors"
	"fmt"
	"strings"

	model "backend/internal/models"
	"backend/internal/repository"

	"gorm.io/gorm"
)

var ErrInvalidPublisher = errors.New("invalid publisher")

func validatePublisher(
	name string,
	address string,
	contactNumber string,
) (string, string, string, error) {
	name = strings.TrimSpace(name)
	address = strings.TrimSpace(address)
	contactNumber = strings.TrimSpace(contactNumber)

	if name == "" {
		return "", "", "", fmt.Errorf(
			"%w: publisher name is required",
			ErrInvalidPublisher,
		)
	}

	if len(name) > 255 {
		return "", "", "", fmt.Errorf(
			"%w: publisher name must not exceed 255 characters",
			ErrInvalidPublisher,
		)
	}

	return name, address, contactNumber, nil
}

func writePublisherAudit(
	tx *gorm.DB,
	userID int,
	actionCode string,
	description string,
	ip string,
	userAgent string,
) error {
	actionID, err := repository.GetAuditActionID(tx, actionCode)
	if err != nil {
		return fmt.Errorf("get audit action ID: %w", err)
	}

	moduleID, err := repository.GetAuditModuleID(tx, "CATALOG")
	if err != nil {
		return fmt.Errorf("get audit module ID: %w", err)
	}

	return repository.InsertAuditLog(
		tx,
		&userID,
		actionID,
		moduleID,
		description,
		ip,
		userAgent,
	)
}

func CreatePublisher(
	db *gorm.DB,
	name string,
	address string,
	contactNumber string,
	userID int,
	ip string,
	userAgent string,
) (*model.Publisher, error) {
	name, address, contactNumber, err := validatePublisher(
		name, address, contactNumber,
	)
	if err != nil {
		return nil, err
	}

	publisher := &model.Publisher{
		PublisherName: name,
		Address:       address,
		ContactNumber: contactNumber,
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := repository.CreatePublisher(tx, publisher); err != nil {
			return err
		}

		return writePublisherAudit(
			tx,
			userID,
			"CREATE",
			fmt.Sprintf(
				"Created publisher: %s (ID: %d)",
				publisher.PublisherName,
				publisher.PublisherID,
			),
			ip,
			userAgent,
		)
	})
	if err != nil {
		return nil, err
	}

	return publisher, nil
}

func GetAllPublishers(
	db *gorm.DB,
	userID int,
	ip string,
	userAgent string,
) ([]model.Publisher, error) {
	var publishers []model.Publisher

	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		publishers, err = repository.GetAllPublishers(tx)
		if err != nil {
			return err
		}

		return writePublisherAudit(
			tx, userID, "READ",
			"Retrieved all publishers", ip, userAgent,
		)
	})
	if err != nil {
		return nil, err
	}

	return publishers, nil
}

func GetPublisherByID(
	db *gorm.DB,
	publisherID int,
	userID int,
	ip string,
	userAgent string,
) (model.Publisher, error) {
	var publisher model.Publisher

	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		publisher, err = repository.GetPublisherByID(tx, publisherID)
		if err != nil {
			return err
		}

		return writePublisherAudit(
			tx,
			userID,
			"READ",
			fmt.Sprintf("Retrieved publisher ID: %d", publisherID),
			ip,
			userAgent,
		)
	})
	if err != nil {
		return model.Publisher{}, err
	}

	return publisher, nil
}

func UpdatePublisher(
	db *gorm.DB,
	publisherID int,
	name string,
	address string,
	contactNumber string,
	userID int,
	ip string,
	userAgent string,
) (model.Publisher, error) {
	name, address, contactNumber, err := validatePublisher(
		name, address, contactNumber,
	)
	if err != nil {
		return model.Publisher{}, err
	}

	var publisher model.Publisher

	err = db.Transaction(func(tx *gorm.DB) error {
		oldPublisher, err := repository.GetPublisherByID(tx, publisherID)
		if err != nil {
			return err
		}

		if err := repository.UpdatePublisher(
			tx, publisherID, name, address, contactNumber,
		); err != nil {
			return err
		}

		publisher, err = repository.GetPublisherByID(tx, publisherID)
		if err != nil {
			return err
		}

		description := fmt.Sprintf(
			"Updated publisher ID %d: name '%s' to '%s'",
			publisherID,
			oldPublisher.PublisherName,
			publisher.PublisherName,
		)

		return writePublisherAudit(
			tx, userID, "UPDATE", description, ip, userAgent,
		)
	})
	if err != nil {
		return model.Publisher{}, err
	}

	return publisher, nil
}

func DeletePublisher(
	db *gorm.DB,
	publisherID int,
	userID int,
	ip string,
	userAgent string,
) error {
	return db.Transaction(func(tx *gorm.DB) error {
		publisher, err := repository.GetPublisherByID(tx, publisherID)
		if err != nil {
			return err
		}

		if err := repository.DeletePublisher(tx, publisherID); err != nil {
			return err
		}

		return writePublisherAudit(
			tx,
			userID,
			"DELETE",
			fmt.Sprintf(
				"Deleted publisher: %s (ID: %d)",
				publisher.PublisherName,
				publisher.PublisherID,
			),
			ip,
			userAgent,
		)
	})
}
