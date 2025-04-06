package service

import (
	"fmt"
	"strings"
)

type AlipayRecord struct {
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

func TransAlipay(records [][]string) []string {
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

		record := AlipayRecord{
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

		date := timeParts[0]
		time := timeParts[1]

		entry := fmt.Sprintf(`
		time: "%s"
		uuid: "%s"
		status: "%s"
		Expenses:Misc                                 %s CNY
		Assets:Alipay                                -%s CNY
		%s * "%s" "%s"
		`,
			date,
			record.Counterparty,
			record.Commodity,
			time,
			record.UUID,
			record.TransactionStatus,
			record.Amount,
			record.Amount,
		)
		result = append(result, entry)
	}
	fmt.Println(result)
	return result
}

func TransWechat(records [][]string) []string {
	var result []string
	if len(records) <= 16 {
		return result
	}

	for i, row := range records[16:] {
		if len(row) < 11 {
			continue
		}
		transactionTime := strings.TrimSpace(row[0])
		timeParts := strings.Split(transactionTime, " ")
		if len(timeParts) < 2 {
			fmt.Printf("Skipping row %d: invalid time format: %s\n", i+24, transactionTime)
			continue
		}
		date := timeParts[0]
		time := timeParts[1]

		// transactionType := strings.TrimSpace(row[4])
		amount := strings.TrimSpace(row[5])
		status := strings.TrimSpace(row[7])

		if status == "已全额退款" || status == "对方已退还" {
			continue
		}

		entry := fmt.Sprintf(`
		%s * "%s" "%s"
    	time: "%s"
		uuid: "%s"
		status: "%s"
		Expenses:Misc                                 %s CNY
		Assets:WeChat                                -%s CNY
		`,
			date,
			strings.TrimSpace(row[2]), // counterparty
			strings.TrimSpace(row[3]), // commodity
			time,
			strings.TrimSpace(row[8]), // uuid
			status,
			amount,
			amount,
		)
		result = append(result, entry)
	}
	fmt.Println(result)
	return result
}
