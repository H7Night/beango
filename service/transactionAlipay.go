package service

import (
	"beango/core"
	"beango/model"
	"fmt"
	"gorm.io/gorm"
	"log"
	"strconv"
	"strings"
)

func TransAlipay(records [][]string, mappings []model.AccountMapping) []string {
	var result []string
	if len(records) <= 24 {
		log.Println("Too few records to process")
		return result
	}
outerLoop:
	for _, row := range records[24:] {
		if len(row) < 12 {
			continue
		}
		commodity := strings.TrimSpace(row[4])
		// 收/支
		transactionType := row[5]

		if transactionType == "不计收支" {
			// 依次检查 commodity 中包含的关键词
			matched := false
			for keyword, mapType := range model.CommodityTypeMap {
				if strings.Contains(commodity, keyword) {
					if mapType == "skip" {
						continue outerLoop
					}
					transactionType = mapType
					matched = true
					break
				}
			}
			if !matched {
				transactionType = "/" // 保持为未知类型
			}
		}

		transactionStatus := strings.TrimSpace(row[8])
		if transactionStatus == "交易关闭" {
			continue
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
		// 支付方式分离，如果有&，选&前面的
		discount := strings.Contains(paymentMethod, "&")
		if discount {
			paymentMethod = strings.Split(paymentMethod, "&")[0]
		}

		record := model.TransactionRecord{
			TransactionTime:   strings.TrimSpace(row[0]),
			TransactionCat:    strings.TrimSpace(row[1]),
			Counterparty:      strings.TrimSpace(row[2]),
			Commodity:         commodity,
			TransactionType:   transactionType,
			Amount:            strings.TrimSpace(row[6]),
			PaymentMethod:     paymentMethod,
			TransactionStatus: transactionStatus,
			Notes:             notes,
			UUID:              strings.TrimSpace(row[9]),
			Discount:          discount,
		}

		db := core.GetDB()
		entry := formatAlipayTransactionEntry(db, record, mappings)
		result = append(result, entry)
		log.Print(result)
	}
	return result
}

func formatAlipayTransactionEntry(db *gorm.DB, record model.TransactionRecord, mappings []model.AccountMapping) string {
	db.Find(&mappings)
	// 默认账户
	expenseAccount := "Expenses:Other"
	incomeAccount := "Income:Other"
	assetAccount := "Assets:Other"
	// 可匹配的字段组合
	combinedText := record.Counterparty + record.Commodity + record.PaymentMethod + record.Notes
	for _, mapping := range mappings {
		if strings.Contains(combinedText, mapping.Keyword) {
			switch mapping.Type {
			case "expense":
				if expenseAccount == "Expenses:Other" {
					expenseAccount = mapping.Account
				}
			case "income":
				if incomeAccount == "Income:Other" {
					incomeAccount = mapping.Account
				}
			case "asset":
				if assetAccount == "Assets:Other" {
					assetAccount = mapping.Account
				}
			}
		}

	}

	date := strings.Split(record.TransactionTime, " ")[0]
	time := strings.Split(record.TransactionTime, " ")[1]
	amount, _ := strconv.ParseFloat(record.Amount, 64)
	commodity := record.Commodity

	// 生成 Beancount 条目
	var entryBuilder strings.Builder
	entryBuilder.WriteString(fmt.Sprintf("%s * \"%s\" \"%s\"\n", date, record.Counterparty, commodity))
	entryBuilder.WriteString(fmt.Sprintf("    time: \"%s\"\n", time))
	entryBuilder.WriteString(fmt.Sprintf("    uuid: \"%s\"\n", record.UUID))
	entryBuilder.WriteString(fmt.Sprintf("    status: \"%s\"\n", record.TransactionStatus))

	switch record.TransactionType {
	case "支出":
		entryBuilder.WriteString(fmt.Sprintf("    %s    %.2f CNY\n", expenseAccount, amount))
		entryBuilder.WriteString(fmt.Sprintf("    %s   -%.2f CNY\n", assetAccount, amount))
	case "收入":
		entryBuilder.WriteString(fmt.Sprintf("    %s   -%.2f CNY\n", incomeAccount, amount))
		entryBuilder.WriteString(fmt.Sprintf("    %s    %.2f CNY\n", assetAccount, amount))
	default:
		// 不计收支或其他类型
		entryBuilder.WriteString(fmt.Sprintf("    undefined    %.2f CNY\n", amount))
		entryBuilder.WriteString(fmt.Sprintf("    undefined   -%.2f CNY\n", amount))
	}
	return entryBuilder.String()
}
