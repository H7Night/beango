package service

import (
	"beango/core"
	"beango/model"
	"fmt"
	"gorm.io/gorm"
	"log"
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
		fmt.Println("Too few records to process")
		return result
	}
	for i, row := range records[24:] {
		if len(row) < 12 {
			continue
		}
		commodity := strings.TrimSpace(row[4])
		// 收/支
		transactionType := row[5]
		if transactionType == "不计收支" {
			if strings.Contains(commodity, "收益发放") {
				transactionType = "收入"
			} else if strings.Contains(commodity, "买入") {
				transactionType = "支出"
			} else if strings.Contains(commodity, "转入") {
				continue
			} else if strings.Contains(commodity, "信用卡还款") {
				transactionType = "支出"
			}
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
		db := core.GetDB()
		entry := formatTransactionEntry(db, record, mappings)
		result = append(result, entry)
	}
	fmt.Println(result)
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
	}
	fmt.Println(result)
	return result
}
func formatTransactionEntry(db *gorm.DB, record model.TransactionRecord, mappings []model.AccountMapping) string {
	// 合并用于匹配的内容字段
	content := record.Counterparty + " " + record.TransactionCat + " " + record.Notes

	// 判断是支出还是收入
	isExpense := record.TransactionType == "支出"
	isIncome := record.TransactionType == "收入"

	// 默认账户
	expenseAccount := "Expenses:Other"
	incomeAccount := "Income:Other"
	assetAccount := "Assets:Other"

	// 查询 expense/income 映射
	if isExpense || isIncome {
		var mappings []model.AccountMapping
		matchType := "expense"
		if isIncome {
			matchType = "income"
		}
		db.Where("type = ?", matchType).Find(&mappings)
		for _, mapping := range mappings {
			if strings.Contains(content, mapping.Keyword) {
				if isExpense {
					expenseAccount = mapping.Account
				} else {
					incomeAccount = mapping.Account
				}
				break
			}
		}
	}

	// 查询资产账户映射（根据支付方式）
	var assetMappings []model.AccountMapping
	db.Where("type = ?", "asset").Find(&assetMappings)
	for _, mapping := range assetMappings {
		if strings.Contains(record.PaymentMethod, mapping.Keyword) {
			assetAccount = mapping.Account
			break
		}
	}

	// 时间与金额处理
	dateTime := strings.Split(record.TransactionTime, " ")
	if len(dateTime) != 2 {
		log.Printf("invalid TransactionTime format: %s", record.TransactionTime)
		return ""
	}
	date, time := dateTime[0], dateTime[1]
	amount := record.Amount
	commodity := record.Commodity

	// 生成 Beancount 条目
	var entry string
	if isExpense {
		entry = fmt.Sprintf(`
			%s * "%s" "%s"
			time: "%s"
			uuid: "%s"
			status: "%s"
			%s				 %s CNY
			%s				-%s CNY
`, date, record.Counterparty, commodity,
			time,
			record.UUID,
			record.TransactionStatus,
			expenseAccount, amount,
			assetAccount, amount)
	} else if isIncome {
		entry = fmt.Sprintf(`
			%s * "%s" "%s"
			time: "%s"
			uuid: "%s"
			status: "%s"
			%s    		-%s CNY
			%s     		 %s CNY
`, date, record.Counterparty, commodity,
			time,
			record.UUID,
			record.TransactionStatus,
			incomeAccount, amount,
			assetAccount, amount)
	} else {
		// 其他类型（如退款、转账等）可根据需要扩展
		entry = fmt.Sprintf("; unsupported transaction type: %s\n", record.TransactionType)
	}
	return entry
}
