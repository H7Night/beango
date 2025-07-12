package model

import (
	"log"
	"time"
)

var mappings []AccountMap

type AccountMap struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"ID"`
	Keyword   string    `gorm:"type:varchar(64);index" json:"keyword"` // 模糊关键词
	Account   string    `gorm:"type:varchar(128)" json:"account"`      // 映射后的账户名
	Type      string    `gorm:"type:varchar(32)" json:"type"`          // 类型: expense / income / asset 等
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// LoadAccountMapFromDB 加载数据库映射库
func LoadAccountMapFromDB() error {
	err := db.Find(&mappings).Error
	if err != nil {
		log.Fatal("无法加载账户映射:", err)
	}
	return err
}

func GetAccountMap() []AccountMap {
	return mappings
}

func CreateAccountMap(accountMap AccountMap) error {
	return db.Create(&accountMap).Error
}

func UpdateAccountMap(id uint64, mapp AccountMap) error {
	return db.Model(&mappings).Where("id = ?", id).Updates(AccountMap{
		Keyword: mapp.Keyword,
		Account: mapp.Account,
		Type:    mapp.Type,
	}).Error
}

func DeleteAccountMap(id uint64) error {
	return db.Where("id=?", id).Delete(&AccountMap{}).Error
}

func GetAllAccountMap() ([]AccountMap, error) {
	var mappings []AccountMap
	err := db.Find(&mappings).Error
	return mappings, err
}
