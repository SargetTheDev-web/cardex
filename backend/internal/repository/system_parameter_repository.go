// internal/repository/system_parameter_repository.go

package repository

import (
	"errors"

	"gorm.io/gorm"
)

type SystemParameter struct {
	ParameterKey   string `gorm:"column:parameter_key" json:"parameter_key"`
	ParameterValue string `gorm:"column:parameter_value" json:"parameter_value"`
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
	var params []SystemParameter

	err := db.
		Table("system_parameter").
		Select("parameter_key, parameter_value").
		Order("parameter_key ASC").
		Find(&params).Error

	if err != nil {
		return nil, err
	}

	return params, nil
}

func UpdateSystemParameter(
	db *gorm.DB,
	key string,
	value string,
) error {
	result := db.
		Table("system_parameter").
		Where("parameter_key = ?", key).
		Update("parameter_value", value)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		var count int64

		err := db.
			Table("system_parameter").
			Where("parameter_key = ?", key).
			Count(&count).Error

		if err != nil {
			return err
		}

		if count == 0 {
			return gorm.ErrRecordNotFound
		}
	}

	return nil
}

func IsSystemParameterNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
