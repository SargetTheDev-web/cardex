// internal/service/enum_service.go

package service

import (
	"fmt"

	"backend/internal/repository"

	"gorm.io/gorm"
)

func GetCoursePrograms(db *gorm.DB) ([]string, error) {
	values, err := repository.GetEnumValues(db, "course_program")
	if err != nil {
		return nil, fmt.Errorf("get course programs: %w", err)
	}

	return values, nil
}
