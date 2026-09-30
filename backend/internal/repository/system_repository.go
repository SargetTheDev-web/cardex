package repository

import "gorm.io/gorm"

type SystemParameter struct {
	ParameterValue string `gorm:"column:parameter_value"`
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

	return param.ParameterValue, err
}
