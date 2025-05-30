package model

import "time"

type BeancountTransaction struct {
	ID                uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	CreateAt          time.Time `gorm:"autoCreateTime" json:"createAt"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	TransactionTime   string    `gorm:"type:varchar(50)" json:"date"`
	UUID              string    `gorm:"type:varchar(64)" json:"uuid"`
	TransactionCat    string    `gorm:"type:varchar(100)" json:"transactionCat"`
	TransactionStatus string    `gorm:"type:varchar(32)" json:"status"`
	Counterparty      string    `gorm:"type:varchar(100)" json:"counterparty"`
	Commodity         string    `gorm:"type:varchar(255)" json:"commodity"`
	TransactionType   string    `gorm:"type:varchar(32)" json:"transactionType"`
	Amount            string    `gorm:"type:decimal(10,2)" json:"amount"`
	PaymentMethod     string    `gorm:"type:varchar(50)" json:"paymentMethod"`
	Notes             string    `gorm:"type:varchar(255)" json:"notes"`
	Source            string    `gorm:"type:varchar(32)" json:"source"`
}
