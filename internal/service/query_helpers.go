package service

import (
	"errors"

	"gorm.io/gorm"
)

type idOnly struct {
	ID uint
}

func existsByQuery(db *gorm.DB, model interface{}, query string, args ...interface{}) (bool, error) {
	var result idOnly
	err := db.Model(model).Select("id").Where(query, args...).Limit(1).Take(&result).Error
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return false, err
}
