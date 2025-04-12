package service

import (
	"beango/model"
	"fmt"
	"gorm.io/gorm"
	"log"
	"strings"
)

type TransactionRecord struct {
	TransactionTime   string
	TransactionCat    string
	Counterparty      string
	Commodity         string
	TransactionType   string
	Amount            string
	PaymentMethod     string
	TransactionStatus string
	Notes             string
	UUID              string
	Discount          bool
}

func TransAlipay(records [][]string, db *gorm.DB) []string {
	var result []string
	if len(records) <= 24 {
		fmt.Println("Too few records to process")
		return result
	}
	for i, row := range records[24:] {
		if len(row) < 12 {
			continue
		}
		// 收/支
		transactionType := row[5]
		if transactionType == "不计收支" {
			transactionType = "/"
		}
		// 收付款方式
		paymentMethod := strings.TrimSpace(row[7])
		if paymentMethod == "" {
			paymentMethod = "余额"
		}
		// 备注
		notes := strings.TrimSpace(row[11])
		if notes == "" {
			notes = "/"
		}

		discount := strings.Contains(paymentMethod, "&")
		if discount {
			paymentMethod = strings.Split(paymentMethod, "&")[0]
		}

		record := TransactionRecord{
			TransactionTime:   strings.TrimSpace(row[0]),
			TransactionCat:    strings.TrimSpace(row[1]),
			Counterparty:      strings.TrimSpace(row[2]),
			Commodity:         strings.TrimSpace(row[4]),
			TransactionType:   transactionType,
			Amount:            strings.TrimSpace(row[6]),
			PaymentMethod:     paymentMethod,
			TransactionStatus: strings.TrimSpace(row[8]),
			Notes:             notes,
			UUID:              strings.TrimSpace(row[9]),
			Discount:          discount,
		}

		// 拆分时间字段
		timeParts := strings.Split(record.TransactionTime, " ")
		if len(timeParts) != 2 {
			fmt.Printf("Skipping row %d: invalid time format: %s\n", i+24, record.TransactionTime)
			continue
		}

		entry := formatTransactionEntry(db, record)
		result = append(result, entry)
	}
	fmt.Println(result)
	return result
}

func TransWechat(records [][]string, db *gorm.DB) []string {
	var result []string
	if len(records) <= 16 {
		return result
	}

	for i, row := range records[16:] {
		if len(row) < 11 {
			continue
		}
		// 获取时间和日期
		transactionTime := strings.TrimSpace(row[0])
		timeParts := strings.Split(transactionTime, " ")
		if len(timeParts) < 2 {
			fmt.Printf("Skipping row %d: invalid time format: %s\n", i+24, transactionTime)
			continue
		}
		transactionType := strings.TrimSpace(row[4])
		if transactionType == "不计收支" {
			transactionType = "/"
		}
		paymentMethod := strings.TrimSpace(row[6])
		if paymentMethod == "" {
			paymentMethod = "零钱"
		}

		// transactionType := strings.TrimSpace(row[4])
		amount := strings.TrimSpace(row[5])
		status := strings.TrimSpace(row[7])
		// 忽略退款
		if status == "已全额退款" || status == "对方已退还" {
			continue
		}
		record := TransactionRecord{
			TransactionTime:   transactionTime,
			TransactionCat:    strings.TrimSpace(row[1]),
			Counterparty:      strings.TrimSpace(row[2]),
			Commodity:         strings.TrimSpace(row[3]),
			TransactionType:   transactionType,
			Amount:            amount,
			PaymentMethod:     paymentMethod,
			TransactionStatus: status,
			Notes:             strings.TrimSpace(row[10]),
			UUID:              strings.TrimSpace(row[8]),
			Discount:          false,
		}
		entry := formatTransactionEntry(db, record)
		result = append(result, entry)
	}
	fmt.Println(result)
	return result
}

func formatTransactionEntry(db *gorm.DB, record TransactionRecord) string {
	timeParts := strings.Split(record.TransactionTime, " ")
	if len(timeParts) != 2 {
		log.Printf("Invalid time format: %s\n", record.TransactionTime)
		return ""
	}
	date := timeParts[0]
	time := timeParts[1]

	mappedAccount := model.ApplyAccountMapping(db, record.PaymentMethod, record.TransactionType)
	mappedCategory := model.ApplyCategoryMapping(db, record.TransactionCat, record.TransactionType)

	amount := record.Amount
	entry := fmt.Sprintf(`
	%s * "%s" "%s"
    time: "%s"
    uuid: "%s"
    status: "%s"
    %s                                  %s CNY
    %s                                 -%s CNY`,
		date,
		record.Counterparty,
		record.Commodity,
		time,
		record.UUID,
		record.TransactionStatus,
		mappedCategory,
		amount,
		mappedAccount,
		amount,
	)

	return entry
}
