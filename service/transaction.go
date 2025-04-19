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

func MatchAccountFromDB(text string, mappings []model.AccountMapping, targetType string, defaultAccount string) string {
	for _, mapping := range mappings {
		if mapping.Type == targetType && strings.Contains(text, mapping.Keyword) {
			return mapping.Account
		}
	}
	return defaultAccount
}

func TransAlipay(records [][]string, mappings []model.AccountMapping) []string {
	var result []string
	if len(records) <= 24 {
		log.Println("Too few records to process")
		return result
	}
	for _, row := range records[24:] {
		if len(row) < 12 {
			continue
		}
		commodity := strings.TrimSpace(row[4])
		// 收/支
		transactionType := row[5]
		// commodity 分类关键词映射
		var commodityTypeMap = map[string]string{
			"还款":     "支出",
			"信用卡还款":  "支出",
			"花呗自动还款": "支出",
			"买入":     "支出",
			"收益发放":   "收入",
			"转入":     "跳过",
		}

		if transactionType == "不计收支" {
			// 依次检查 commodity 中包含的关键词
			matched := false
			for keyword, inferredType := range commodityTypeMap {
				if strings.Contains(commodity, keyword) {
					if inferredType == "跳过" {
						continue
					}
					transactionType = inferredType
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
		entry := formatTransactionEntry(db, record, mappings)
		result = append(result, entry)
		log.Print(result)
	}
	return result
}

func TransWechat(records [][]string, mappings []model.AccountMapping) []string {
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
			log.Printf("Skipping row %d: invalid time format: %s\n", i+24, transactionTime)
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

		amount := strings.TrimSpace(row[5])
		status := strings.TrimSpace(row[7])
		// 忽略退款
		if status == "已全额退款" || status == "对方已退还" {
			continue
		}
		record := model.TransactionRecord{
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

		db := core.GetDB()
		entry := formatTransactionEntry(db, record, mappings)
		result = append(result, entry)
		print(result)
	}
	return result
}

func formatTransactionEntry(db *gorm.DB, record model.TransactionRecord, mappings []model.AccountMapping) string {
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
