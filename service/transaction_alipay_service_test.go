package service

import (
	"os"
	"strings"
	"testing"
)

// alipayRepaymentRow 模拟支付宝 CSV 中"招商银行 信用卡还款"（从余额宝还信用卡）的记录行
func alipayRepaymentRow() []string {
	return []string{
		"2026-08-05 17:15:06",                    // transactionTime
		"信用借还",                                // transactionCat
		"招商银行",                                // counterparty
		"/",                                      // 4th col (unused)
		"信用卡还款",                              // commodity
		"不计收支",                                // transactionType
		"1247.80",                                // amount
		"余额宝",                                  // paymentMethod
		"还款成功",                                // transactionStatus
		"2026080500003001520093859760",           // uuid
		"",                                       // 11th col (unused)
		"/",                                      // notes
	}
}

func TestTransAlipayRepayment(t *testing.T) {
	// go test 的工作目录是包目录 (service/)，配置在项目根
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd 失败: %v", err)
	}
	if err := os.Chdir(".."); err != nil {
		t.Fatalf("Chdir 失败: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	// TransAlipay 要求 len(records) > 24，首行被跳过
	var records [][]string
	for i := 0; i < 25; i++ {
		records = append(records, alipayRepaymentRow())
	}

	entries, count, err := TransAlipay(records)
	if err != nil {
		t.Fatalf("TransAlipay 失败: %v", err)
	}
	if len(entries) != 24 {
		t.Fatalf("期望 24 条记录，实际 %d", len(entries))
	}
	if count[2] != 24 {
		t.Fatalf("期望 24 笔转账，实际 %d", count[2])
	}

	entry := entries[0]
	if !strings.Contains(entry, "Liabilities:CMBCreditCard:2035") {
		t.Errorf("还款目标应为 Liabilities:CMBCreditCard:2035，实际:\n%s", entry)
	}
	if !strings.Contains(entry, "Assets:AliPay:Balance") {
		t.Errorf("还款来源应为 Assets:AliPay:Balance，实际:\n%s", entry)
	}
	if strings.Contains(entry, "Assets:CMB:3229") {
		t.Errorf("不应误匹配为 Assets:CMB:3229，实际:\n%s", entry)
	}
}
