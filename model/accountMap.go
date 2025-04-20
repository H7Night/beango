package model

import (
	"gorm.io/gorm"
	"log"
	"time"
)

var mappings []AccountMapping

type AccountMapping struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	Keyword   string    `gorm:"type:varchar(64);index"` // 模糊关键词
	Account   string    `gorm:"type:varchar(128)"`      // 映射后的账户名
	Type      string    `gorm:"type:varchar(32)"`       // 类型: expense / income / asset 等
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// LoadAccountMappingsFromDB 加载数据库映射库
func LoadAccountMappingsFromDB() error {
	err := GetDB().Find(&mappings).Error
	if err != nil {
		log.Fatal("无法加载账户映射:", err)
	}
	return err
}

func GetAccountMappings() []AccountMapping {
	return mappings
}

func CreateAccountMapping(db *gorm.DB, keyword, account, mappingType string) error {
	return db.Create(&AccountMapping{Keyword: keyword, Account: account, Type: mappingType}).Error
}

func UpdateAccountMapping(db *gorm.DB, id uint, keyword, account, mappingType string) error {
	return db.Model(&AccountMapping{}).Where("id=?", id).Updates(AccountMapping{Keyword: keyword, Account: account, Type: mappingType}).Error
}

func DeleteAccountMapping(db *gorm.DB, id uint) error {
	return db.Delete(&AccountMapping{}, id).Error
}

func GetAllAccountMapping(db *gorm.DB) ([]AccountMapping, error) {
	var mappings []AccountMapping
	err := db.Find(&mappings).Error
	return mappings, err
}
