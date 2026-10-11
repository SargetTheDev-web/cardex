// internal/repository/enum_repository.go

package repository

import "gorm.io/gorm"

func GetEnumValues(db *gorm.DB, enumType string) ([]string, error) {
	var values []string

	err := db.Raw(`
		SELECT e.enumlabel
		FROM pg_enum e
		JOIN pg_type t ON t.oid = e.enumtypid
		JOIN pg_namespace n ON n.oid = t.typnamespace
		WHERE t.typname = ?
		  AND n.nspname = current_schema()
		ORDER BY e.enumsortorder
	`, enumType).Scan(&values).Error

	if err != nil {
		return nil, err
	}

	return values, nil
}
