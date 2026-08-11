package service

import (
	"beango/model"
	"beango/utils"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
)

func TransAlipay(records [][]string) ([]string, [5]int, error) {
	var result []string
	// 重置计数器
	count = [5]int{0, 0, 0, 0}
	if len(records) <= 24 {
		log.Println("导入文件不符合支付宝格式")
		return nil, [5]int{}, errors.New("导入文件不符合支付宝格式")
	}
outerLoop:
	for _, row := range records[1:] {
		// 前12行为不必要数据
		if len(row) < 12 {
			continue
		}

		// 提取字段 + Trim
		transactionTime := strings.TrimSpace(row[0])
		transactionCat := strings.TrimSpace(row[1])
		counterparty := strings.TrimSpace(row[2])
		commodity := strings.TrimSpace(row[4])
		transactionType := strings.TrimSpace(row[5])
		amount := strings.TrimSpace(row[6])
		paymentMethod := strings.TrimSpace(row[7])
		transactionStatus := strings.TrimSpace(row[8])
		uuid := strings.TrimSpace(row[9])
		notes := strings.TrimSpace(row[11])

		commodityMap, _ := model.LoadCommodityMap(model.CommodityMapPath())

		if transactionType == "不计收支" {
			// 默认视为转账，除非匹配到 skip
			transactionType = "转账"
			matched := false
			for keyword, mapType := range commodityMap {
				if strings.Contains(commodity, keyword) {
					if mapType == "skip" {
						count[4]++
						utils.LogConvert("skip", row)
						continue outerLoop //不记录该数据
					}
					transactionType = mapType
					matched = true
					break
				}
			}
			// 如果 commodity 没匹配到，检查 transactionCat 或 commodity 里的关键词
			if !matched {
				if strings.Contains(transactionCat, "转入") || strings.Contains(transactionCat, "转出") ||
				   strings.Contains(transactionCat, "理财") || strings.Contains(transactionCat, "还款") ||
				   strings.Contains(commodity, "转入") || strings.Contains(commodity, "转出") ||
				   strings.Contains(commodity, "还款") || strings.Contains(commodity, "买入") {
					transactionType = "转账"
				} else {
					transactionType = "undefined"
				}
			}
		}
		// 交易状态
		if transactionStatus == "交易关闭" || (strings.Contains(transactionStatus, "退款") && !strings.Contains(transactionStatus, "(")) {
			utils.LogConvert("skip", row)
			continue
		}
		// 支付方式分离，如果有&，选&前面的
		discount := strings.Contains(paymentMethod, "&")
		if discount {
			paymentMethod = strings.Split(paymentMethod, "&")[0]
		} else if paymentMethod == "" {
			paymentMethod = "余额"
		}
		// 备注
		if notes == "" {
			notes = "/"
		}

		record := model.BeancountTransaction{
			TransactionTime:   transactionTime,
			TransactionCat:    transactionCat,
			Counterparty:      counterparty,
			Commodity:         commodity,
			TransactionType:   transactionType,
			Amount:            amount,
			PaymentMethod:     paymentMethod,
			TransactionStatus: transactionStatus,
			Notes:             notes,
			UUID:              uuid,
			Source:            "alipay",
		}

		entry := formatAlipayTransactionEntry(record)
		result = append(result, entry)
	}
	return result, count, nil
}

func formatAlipayTransactionEntry(record model.BeancountTransaction) string {

	accountMap := model.GetAccountMap()
	// 默认账户（由配置决定，未匹配时兜底）
	defaultExpense := model.DefaultExpenseAccount()
	defaultIncome := model.DefaultIncomeAccount()
	defaultAsset := model.DefaultAssetAccount()
	expenseAccount := defaultExpense
	incomeAccount := defaultIncome
	assetAccount := defaultAsset
	fromAccount := defaultAsset
	toAccount := defaultAsset

	// 1. 匹配支付方式 (通常是支出/收入的资金来源/去向，或者是转账的资金来源)
	for _, mapping := range accountMap {
		if mapping.Type == "asset" && strings.Contains(record.PaymentMethod, mapping.Keyword) {
			assetAccount = mapping.Account
			fromAccount = mapping.Account
			break
		}
	}

	// 2. 优先匹配交易对方 (更精确的 Counterparty 优先于商品/分类关键词)
	for _, mapping := range accountMap {
		if strings.Contains(record.Counterparty, mapping.Keyword) || strings.Contains(mapping.Keyword, record.Counterparty) {
			switch mapping.Type {
			case "expense":
				if expenseAccount == defaultExpense {
					expenseAccount = mapping.Account
				}
			case "income":
				if incomeAccount == defaultIncome {
					incomeAccount = mapping.Account
				}
			case "asset":
				if toAccount == defaultAsset {
					toAccount = mapping.Account
				}
			}
		}
	}

	// 3. 匹配商品信息 + 交易分类 (用于识别支出分类，或转账的目标账户)
	combinedDest := record.Counterparty + record.Commodity + record.Notes + record.TransactionCat
	for _, mapping := range accountMap {
		if strings.Contains(combinedDest, mapping.Keyword) {
			switch mapping.Type {
			case "expense":
				if expenseAccount == defaultExpense {
					expenseAccount = mapping.Account
				}
			case "income":
				if incomeAccount == defaultIncome {
					incomeAccount = mapping.Account
				}
			case "asset":
				if toAccount == defaultAsset {
					toAccount = mapping.Account
				}
			}
		}
	}

	// 4. 特殊处理：如果识别为转账，且 from/to 还是默认值，尝试根据交易分类进一步识别
	if record.TransactionType == "转账" {
		isRepayment := strings.Contains(record.TransactionCat, "还款") || strings.Contains(record.Commodity, "还款")

		if isRepayment {
			// 还款情况：资金来源 (from) 是支付方式，目标 (to) 是还款对象（如花呗/信用卡）
			// 还款对象以 commodity + 交易分类为准，覆盖 counterparty 反向包含导致的误匹配
			// （如 counterparty "招商银行" 会被 "招商银行(3229)" 反向包含匹配成储蓄卡）
			repaymentTarget := record.Commodity + record.TransactionCat
			for _, mapping := range accountMap {
				if mapping.Type == "asset" && strings.Contains(repaymentTarget, mapping.Keyword) {
					toAccount = mapping.Account
					break
				}
			}
			// 兜底：commodity/交易分类未匹配到还款对象时，退回 counterparty 匹配
			if toAccount == defaultAsset {
				for _, mapping := range accountMap {
					if mapping.Type == "asset" && (strings.Contains(record.Commodity, mapping.Keyword) || strings.Contains(record.Counterparty, mapping.Keyword) || strings.Contains(mapping.Keyword, record.Counterparty)) {
						toAccount = mapping.Account
						break
					}
				}
			}
		} else if strings.Contains(record.TransactionCat, "转入") || strings.Contains(record.Commodity, "转入") {
			// 这种情况下，paymentMethod 通常是外部账户 (from)，商品信息里包含的是内部账户 (to)
			// 如果 toAccount 还没匹配到，再次尝试从 commodity 匹配 "余额宝" 等
			if toAccount == defaultAsset {
				for _, mapping := range accountMap {
					if mapping.Type == "asset" && (strings.Contains(record.Commodity, mapping.Keyword) || strings.Contains(record.Counterparty, mapping.Keyword) || strings.Contains(mapping.Keyword, record.Counterparty)) {
						toAccount = mapping.Account
						break
					}
				}
			}
		} else if strings.Contains(record.TransactionCat, "转出") || strings.Contains(record.Commodity, "转出") {
			// 这种情况下，paymentMethod 通常是内部账户 (from)，商品信息里包含的是外部账户 (to)
			if fromAccount == defaultAsset {
				for _, mapping := range accountMap {
					if mapping.Type == "asset" && (strings.Contains(record.Commodity, mapping.Keyword) || strings.Contains(record.Counterparty, mapping.Keyword) || strings.Contains(mapping.Keyword, record.Counterparty)) {
						fromAccount = mapping.Account
						break
					}
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
		count[0]++
		entryBuilder.WriteString(fmt.Sprintf("    %s    %.2f CNY\n", expenseAccount, amount))
		entryBuilder.WriteString(fmt.Sprintf("    %s   -%.2f CNY\n", assetAccount, amount))
		utils.LogConvert("success", record)
	case "收入":
		count[1]++
		entryBuilder.WriteString(fmt.Sprintf("    %s    %.2f CNY\n", assetAccount, amount))
		entryBuilder.WriteString(fmt.Sprintf("    %s   -%.2f CNY\n", incomeAccount, amount))
		utils.LogConvert("success", record)
	case "转账":
		count[2]++
		entryBuilder.WriteString(fmt.Sprintf("    %s    %.2f CNY\n", toAccount, amount))
		entryBuilder.WriteString(fmt.Sprintf("    %s   -%.2f CNY\n", fromAccount, amount))
		utils.LogConvert("success", record)
	default: // 无法解析的数据
		count[3]++
		entryBuilder.WriteString(fmt.Sprintf("    undefined    %.2f CNY\n", amount))
		entryBuilder.WriteString(fmt.Sprintf("    undefined   -%.2f CNY\n", amount))
		utils.LogConvert("undefined", record)
	}
	return entryBuilder.String()
}
