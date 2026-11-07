// internal/repository/system_parameter_repository.go

package repository

import (
	"gorm.io/gorm"
)

type SystemParameter struct {
	ParameterID    int64  `gorm:"column:parameter_id" json:"parameter_id"`
	ParentID       *int64 `gorm:"column:parent_id" json:"parent_id"`
	ParameterKey   string `gorm:"column:parameter_key" json:"parameter_key"`
	ParameterValue string `gorm:"column:parameter_value" json:"parameter_value"`
	DataType       string `gorm:"column:data_type" json:"data_type"`
	Description    string `gorm:"column:description" json:"description"`
	Path           string `gorm:"column:path" json:"path"`
}

func GetSystemParameter(
	db *gorm.DB,
	key string,
) (string, error) {
	var param SystemParameter

	err := db.
		Table("system_parameter").
		Select("parameter_value").
		Where("parameter_key = ?", key).
		First(&param).Error

	if err != nil {
		return "", err
	}

	return param.ParameterValue, nil
}

func GetAllSystemParameters(
	db *gorm.DB,
) ([]SystemParameter, error) {
	params := make([]SystemParameter, 0)

	err := db.Raw(`
		WITH RECURSIVE parameter_tree AS (
			-- Anchor: root-level parameters
			SELECT
				parameter_id,
				parent_id,
				parameter_key,
				parameter_value,
				data_type,
				description,
				0 AS level,
				parameter_key::TEXT AS path
			FROM system_parameter
			WHERE parent_id IS NULL

			UNION ALL

			-- Recursive: child parameters
			SELECT
				p.parameter_id,
				p.parent_id,
				p.parameter_key,
				p.parameter_value,
				p.data_type,
				p.description,
				pt.level + 1,
				(pt.path || ' -> ' || p.parameter_key)::TEXT AS path
			FROM system_parameter p
			INNER JOIN parameter_tree pt
				ON p.parent_id = pt.parameter_id
		)
		SELECT
			parameter_id,
			parent_id,
			LPAD('', level * 4, ' ') || parameter_key AS parameter_key,
			parameter_value,
			data_type,
			description,
			path
		FROM parameter_tree
		ORDER BY path ASC
	`).Scan(&params).Error

	if err != nil {
		return nil, err
	}

	return params, nil
}
