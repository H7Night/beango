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

func TransWechat(records [][]string, mappings []model.AccountMapping) []string {
	var result []string
	if len(records) <= 16 {
		return result
	}

	db := core.GetDB()

	for i, row := range records[16:] {
		record, skip, reason := parseWechatRow(i+17, row)
		if skip {
			log.Printf("Skipping row %d: %s", i+17, reason)
			continue
		}

		entry := formatWechatTransactionEntry(db, record, mappings)
		result = append(result, entry)
		log.Print(result)
	}
	return result
}

func parseWechatRow(rowIndex int, row []string) (model.TransactionRecord, bool, string) {
	if len(row) < 11 {
		return model.TransactionRecord{}, true, "incomplete row"
	}

	transactionTime := strings.TrimSpace(row[0])
	timeParts := strings.Split(transactionTime, " ")
	if len(timeParts) < 2 {
		return model.TransactionRecord{}, true, fmt.Sprintf("invalid time format: %s", transactionTime)
	}

	transactionType := strings.TrimSpace(row[4])
	paymentMethod := strings.TrimSpace(row[6])
	if paymentMethod == "" {
		paymentMethod = "零钱"
	}
	commodity := strings.TrimSpace(row[3])
	status := strings.TrimSpace(row[7])

	// 跳过退款类交易
	if status == "已全额退款" || status == "对方已退还" {
		return model.TransactionRecord{}, true, "refund transaction"
	}

	// 默认推断类型
	if transactionType == "不计收支" {
		for keyword, inferredType := range model.CommodityTypeMap {
			if strings.Contains(commodity, keyword) {
				if inferredType == "跳过" {
					return model.TransactionRecord{}, true, fmt.Sprintf("keyword '%s' matches skip type", keyword)
				}
				transactionType = inferredType
				break
			}
		}
	}

	// 特殊交易类型强制设定为转账
	transactionCat := strings.TrimSpace(row[1])
	if transactionCat == "零钱提现" || transactionCat == "零钱充值" {
		transactionType = "转账"
	}

	amount := strings.TrimPrefix(strings.TrimSpace(row[5]), "¥")

	return model.TransactionRecord{
		TransactionTime:   transactionTime,
		TransactionCat:    transactionCat,
		Counterparty:      strings.TrimSpace(row[2]),
		Commodity:         commodity,
		TransactionType:   transactionType,
		Amount:            amount,
		PaymentMethod:     paymentMethod,
		TransactionStatus: status,
		Notes:             strings.TrimSpace(row[10]),
		UUID:              strings.TrimSpace(row[8]),
		Discount:          false,
	}, false, ""
}

func formatWechatTransactionEntry(db *gorm.DB, record model.TransactionRecord, mappings []model.AccountMapping) string {
	db.Find(&mappings)

	// 默认账户
	expenseAccount := "Expenses:Other"
	incomeAccount := "Income:Other"
	assetAccount := "Assets:Other"
	fromAccount := "Assets:Other"
	toAccount := "Assets:Other"

	// 可匹配的字段组合
	combinedText := record.Counterparty + record.Commodity + record.TransactionStatus + record.PaymentMethod + record.Notes

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

	// 转账类型特殊账户匹配逻辑
	if record.TransactionType == "转账" {
		// 匹配"零钱"账户作为转账一方
		for _, mapping := range mappings {
			if mapping.Type != "asset" {
				continue
			}
			// 匹配 "零钱" 关键词
			if strings.Contains("零钱", mapping.Keyword) {
				if record.TransactionCat == "零钱提现" {
					fromAccount = mapping.Account
				} else if record.TransactionCat == "零钱充值" {
					toAccount = mapping.Account
				}
			}

			// 匹配 Counterparty 关键词
			if strings.Contains(record.Counterparty, mapping.Keyword) {
				if record.TransactionCat == "零钱提现" {
					toAccount = mapping.Account
				} else if record.TransactionCat == "零钱充值" {
					fromAccount = mapping.Account
				}
			}
		}
	}

	// 日期与时间
	date := strings.Split(record.TransactionTime, " ")[0]
	time := strings.Split(record.TransactionTime, " ")[1]

	// 金额
	amount, _ := strconv.ParseFloat(record.Amount, 64)

	// 描述信息
	var commodityNote string
	if record.Notes == "/" || record.Notes == "" {
		commodityNote = record.Commodity
	} else {
		commodityNote = record.Commodity + record.Notes
	}

	// 构造 Beancount 条目
	var entryBuilder strings.Builder
	entryBuilder.WriteString(fmt.Sprintf("%s * \"%s\" \"%s\"\n", date, record.Counterparty, commodityNote))
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
	case "转账":
		entryBuilder.WriteString(fmt.Sprintf("    %s   -%.2f CNY\n", fromAccount, amount))
		entryBuilder.WriteString(fmt.Sprintf("    %s    %.2f CNY\n", toAccount, amount))
	default:
		entryBuilder.WriteString(fmt.Sprintf("    undefined    %.2f CNY\n", amount))
		entryBuilder.WriteString(fmt.Sprintf("    undefined   -%.2f CNY\n", amount))
	}

	return entryBuilder.String()
}
