package model

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
)

type BeangoConfig struct {
	ID          uint64    `gorm:"primary_key;auto_increment" json:"id"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	ConfigKey   string    `gorm:"type:varchar(255)" json:"config_key"`
	ConfigValue string    `gorm:"type:varchar(255)" json:"config_value"`
	Note        string    `gorm:"type:varchar(255)" json:"note"`
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

func GetConfigString(key, defaultVal string) string {
	var cfg BeangoConfig
	err := db.Where("config_key = ?", key).First(&cfg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return defaultVal
		}
		fmt.Printf("读取配置失败 key=%s: %v\n", key, err)
		return defaultVal
	}
	return cfg.ConfigValue
}

func GetConfigBool(key string, defaultVal bool) bool {
	val := GetConfigString(key, "")
	if val == "" {
		return defaultVal
	}
	res, err := strconv.ParseBool(val)
	if err != nil {
		fmt.Printf("布尔配置解析失败 key=%s: %v\n", key, err)
		return defaultVal
	}
	return res
}

func GetConfigInt(key string, defaultVal int) int {
	val := GetConfigString(key, "")
	if val == "" {
		return defaultVal
	}
	res, err := strconv.Atoi(val)
	if err != nil {
		fmt.Printf("整数配置解析失败 key=%s: %v\n", key, err)
		return defaultVal
	}
	return res
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
