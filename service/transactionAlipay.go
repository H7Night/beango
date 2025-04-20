package service

import (
	"beango/model"
	"fmt"
	"log"
	"strconv"
	"strings"
)

var commodityTypeMap = model.CommodityTypeMap

func TransAlipay(records [][]string) []string {
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
		transactionType := row[5] // 收支

		if transactionType == "不计收支" {
			// 检查 commodity 中包含的关键词
			matched := false
			for keyword, mapType := range commodityTypeMap {
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
		// 交易状态
		transactionStatus := strings.TrimSpace(row[8])
		if transactionStatus == "交易关闭" {
			continue
		}
		// 收付款方式
		paymentMethod := strings.TrimSpace(row[7])
		// 支付方式分离，如果有&，选&前面的
		discount := strings.Contains(paymentMethod, "&")
		if discount {
			paymentMethod = strings.Split(paymentMethod, "&")[0]
		} else if paymentMethod == "" {
			paymentMethod = "余额"
		}
		// 备注
		notes := strings.TrimSpace(row[11])
		if notes == "" {
			notes = "/"
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

		entry := formatAlipayTransactionEntry(record)
		result = append(result, entry)
		log.Print(result)
	}
	return result
}

func formatAlipayTransactionEntry(record model.TransactionRecord) string {
	mappings := model.GetAccountMappings()
	// 默认账户
	expenseAccount := "Expenses:Other"
	incomeAccount := "Income:Other"
	assetAccount := "Assets:Other"
	// 可匹配的字段组合(交易对方+商品信息+付款方式+备注)
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
	default: // 无法解析的数据
		entryBuilder.WriteString(fmt.Sprintf("    undefined    %.2f CNY\n", amount))
		entryBuilder.WriteString(fmt.Sprintf("    undefined   -%.2f CNY\n", amount))
	}
	return entryBuilder.String()
}
