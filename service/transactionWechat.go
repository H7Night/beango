package service

import (
	"beango/model"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
)

func TransWechat(records [][]string) ([]string, error) {
	var result []string
	if len(records) <= 16 {
		return nil, errors.New("too few records to process")
	}

	for _, row := range records[16:] {
		record, skip := parseWechatRow(row)
		if skip {
			continue
		}

		entry := formatWechatTransactionEntry(record)
		result = append(result, entry)
		//log.Print(result)
	}
	return result, nil
}

func parseWechatRow(row []string) (model.TransactionRecord, bool) {
	if len(row) < 11 {
		log.Printf("row too short: %s", row)
		return model.TransactionRecord{}, true
	}

	transactionTime := strings.TrimSpace(row[0])
	timeParts := strings.Split(transactionTime, " ")
	if len(timeParts) < 2 {
		log.Printf("invalid time format: %s", transactionTime)
		return model.TransactionRecord{}, true
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
		log.Printf("invalid status: %s", row)
		return model.TransactionRecord{}, true
	}

	if transactionType == "不计收支" {
		for keyword, inferredType := range model.CommodityTypeMap {
			if strings.Contains(commodity, keyword) {
				if inferredType == "skip" {
					log.Printf("skip commodity: %s", commodity)
					return model.TransactionRecord{}, true
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
	// 金额
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
	}, false
}

func formatWechatTransactionEntry(record model.TransactionRecord) string {
	mappings := model.GetAccountMappings()

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
	default: // 无法解析的数据
		entryBuilder.WriteString(fmt.Sprintf("    undefined    %.2f CNY\n", amount))
		entryBuilder.WriteString(fmt.Sprintf("    undefined   -%.2f CNY\n", amount))
	}

	return entryBuilder.String()
}
