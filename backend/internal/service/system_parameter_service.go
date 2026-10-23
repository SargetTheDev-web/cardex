// internal/service/system_parameter_service.go

package service

import (
	"errors"
	"fmt"
	"unicode/utf8"

	"backend/internal/repository"

	"gorm.io/gorm"
)

const systemParameterModuleCode = "SYSTEM_PARAMETER"

const (
	AuditActionViewAll = "SYSTEM_PARAMETER_VIEW_ALL"
	AuditActionView    = "SYSTEM_PARAMETER_VIEW"
	AuditActionUpdate  = "SYSTEM_PARAMETER_UPDATE"
)

type SystemParameterService struct {
	db *gorm.DB
}

func NewSystemParameterService(
	db *gorm.DB,
) *SystemParameterService {
	return &SystemParameterService{
		db: db,
	}
}

func (s *SystemParameterService) writeAudit(
	tx *gorm.DB,
	userID *int,
	actionCode string,
	description string,
	ip string,
	userAgent string,
) error {
	actionID, err := repository.GetAuditActionID(
		tx,
		actionCode,
	)
	if err != nil {
		return fmt.Errorf("resolve audit action: %w", err)
	}

	moduleID, err := repository.GetAuditModuleID(
		tx,
		systemParameterModuleCode,
	)
	if err != nil {
		return fmt.Errorf("resolve audit module: %w", err)
	}

	if err := repository.InsertAuditLog(
		tx,
		userID,
		actionID,
		moduleID,
		description,
		ip,
		userAgent,
	); err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}

	return nil
}

func (s *SystemParameterService) GetAll(
	userID *int,
	ip string,
	userAgent string,
) ([]repository.SystemParameter, error) {
	var params []repository.SystemParameter

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var err error

		params, err = repository.GetAllSystemParameters(tx)
		if err != nil {
			return fmt.Errorf("retrieve system parameters: %w", err)
		}

		return s.writeAudit(
			tx,
			userID,
			AuditActionViewAll,
			"Retrieved all system parameters",
			ip,
			userAgent,
		)
	})

	if err != nil {
		return nil, err
	}

	return params, nil
}

func (s *SystemParameterService) Get(
	key string,
	userID *int,
	ip string,
	userAgent string,
) (string, error) {
	var value string

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var err error

		value, err = repository.GetSystemParameter(tx, key)
		if err != nil {
			return err
		}

		return s.writeAudit(
			tx,
			userID,
			AuditActionView,
			"Retrieved system parameter: "+key,
			ip,
			userAgent,
		)
	})

	if err != nil {
		return "", err
	}

	return value, nil
}

func (s *SystemParameterService) Update(
	key string,
	value string,
	userID *int,
	ip string,
	userAgent string,
) error {
	if key == "" || len(key) > 100 {
		return errors.New("invalid system parameter key")
	}

	if !utf8.ValidString(value) ||
		utf8.RuneCountInString(value) > 5000 {
		return errors.New("invalid system parameter value")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := repository.UpdateSystemParameter(
			tx,
			key,
			value,
		); err != nil {
			return err
		}

		return s.writeAudit(
			tx,
			userID,
			AuditActionUpdate,
			"Updated system parameter: "+key,
			ip,
			userAgent,
		)
	})
}
