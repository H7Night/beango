package model

import (
	"errors"
	"gorm.io/gorm"
	"time"
)

type BeangoConfig struct {
	ID          uint64    `gorm:"primary_key;auto_increment" json:"id"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	ConfigKey   string    `gorm:"type:varchar(255)" json:"config_key"`
	ConfigValue string    `gorm:"type:varchar(255)" json:"config_value"`
}

func GetBeangoConfigValue(key string) (string, error) {
	var beangoConfig BeangoConfig
	err := db.Where("config_key = ?", key).First(&beangoConfig).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return beangoConfig.ConfigValue, nil
}

func GetAllBeangoConfig() ([]BeangoConfig, error) {
	var beangoConfigs []BeangoConfig
	err := db.Find(&beangoConfigs).Error
	return beangoConfigs, err
}

func CreateBeangoConfig(config BeangoConfig) error {
	return db.Create(&config).Error
}

func UpdateBeangoConfig(id uint64, config BeangoConfig) error {
	return db.Model(&BeangoConfig{}).Where("id = ?", id).Updates(BeangoConfig{
		ConfigKey:   config.ConfigKey,
		ConfigValue: config.ConfigValue,
	}).Error
}
func DeleteBeangoConfig(id uint64) error {
	return db.Model(&BeangoConfig{}).Where("id = ?", id).Delete(&BeangoConfig{}).Error
}
