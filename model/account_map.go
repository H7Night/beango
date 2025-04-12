package model

import (
	"gorm.io/gorm"
	"log"
	"strings"
	"time"
)

type AccountMapping struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	Keyword   string    `gorm:"type:varchar(64);index"` // 模糊关键词
	Account   string    `gorm:"type:varchar(128)"`      // 映射后的账户名
	Type      string    `gorm:"type:varchar(32);index"` // 映射类型: payment / expense / income 等
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func ApplyAccountMapping(db *gorm.DB, paymentMethod, transactionType string) string {
	var mappings []AccountMapping
	if err := db.Find(&mappings).Error; err != nil {
		log.Printf("Failed to load account mappings: %v", err)
		return "Assets:Alipay"
	}

	paymentMethod = strings.TrimSpace(paymentMethod)

	for _, m := range mappings {
		keyword := strings.TrimSpace(m.Keyword)
		if strings.Contains(paymentMethod, keyword) {
			log.Printf("Matched keyword: %s -> %s", keyword, m.Account)
			return m.Account
		}
	}

	return "Assets:Other"
}

func ApplyCategoryMapping(db *gorm.DB, category, paymentMethod string) string {
	var mappings []AccountMapping
	if err := db.Where("type in ?", []string{"expense", "income"}).Find(&mappings).Error; err != nil {
		log.Printf("Failed to load account mappings: %v", err)
		return fallbackCategory(paymentMethod)
	}
	for _, m := range mappings {
		if strings.Contains(category, m.Keyword) || strings.Contains(paymentMethod, m.Keyword) {
			return m.Account
		}
	}
	return fallbackCategory(paymentMethod)
}

func fallbackCategory(typ string) string {
	if strings.Contains(typ, "收入") {
		return "Income:Other"
	}
	return "Expenses:Other"
}

func CreateAccountMapping(db *gorm.DB, keyword, account string) error {
	return db.Create(&AccountMapping{Keyword: keyword, Account: account}).Error
}

func UpdateAccountMapping(db *gorm.DB, id uint, keyword, account string) error {
	return db.Model(&AccountMapping{}).Where("id=?", id).Updates(AccountMapping{Keyword: keyword, Account: account}).Error
}

func DeleteAccountMapping(db *gorm.DB, id uint) error {
	return db.Delete(&AccountMapping{}, id).Error
}

func GetAllAccountMapping(db *gorm.DB) ([]AccountMapping, error) {
	var mappings []AccountMapping
	err := db.Find(&mappings).Error
	return mappings, err
}
