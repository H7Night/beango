package model

import "time"

type BeancountTransaction struct {
	ID                uint      `json:"id"`
	CreateAt          time.Time `json:"createAt"`
	UpdatedAt         time.Time `json:"updated_at"`
	TransactionTime   string    `json:"date"`
	UUID              string    `json:"uuid"`
	TransactionCat    string    `json:"transactionCat"`
	TransactionStatus string    `json:"status"`
	Counterparty      string    `json:"counterparty"`
	Commodity         string    `json:"commodity"`
	TransactionType   string    `json:"transactionType"`
	Amount            string    `json:"amount"`
	PaymentMethod     string    `json:"paymentMethod"`
	Notes             string    `json:"notes"`
	Source            string    `json:"source"`
}
