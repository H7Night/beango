package model

import "time"

type ImportTranscation struct {
	ID       uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Date     string    `gorm:"type:varchar(20)" json:"date"`
	Time     string    `gorm:"type:varchar(20)" json:"time"`
	UUID     string    `gorm:"type:varchar(64)" json:"uuid"`
	Status   string    `gorm:"type:varchar(32)" json:"status"`
	Amount   string    `gorm:"type:decimal(10,2)" json:"amount"`
	Account  string    `gorm:"type:varchar(128)" json:"account"`
	IsSync   bool      `gorm:"default:false" json:"isSync"`
	Source   string    `gorm:"type:varchar(20)" json:"source"`
	CreateAt time.Time `gorm:"autoCreateTime" json:"createAt"`
}
