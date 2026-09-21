package service

import (
	"fmt"
	"strings"
)

// Diagnostic 记录导入过程中被跳过的异常行，便于人工核查。
// Row 是「清洗后」记录序列的下标（含表头，1-based），非原始文件行号。
type Diagnostic struct {
	Source string   `json:"source"` // alipay | wechat
	Row    int      `json:"row"`
	Reason string   `json:"reason"`
	Raw    []string `json:"raw"`
}

// TransResult 是一次账单转换的完整结果。
type TransResult struct {
	Entries     []string
	Count       [5]int // 支出、收入、转账、undefined、skip
	Diagnostics []Diagnostic
}

// bucketOf 将交易类型映射到 Count 下标：支出 0、收入 1、转账 2、其它 3。
func bucketOf(transactionType string) int {
	switch transactionType {
	case "支出":
		return 0
	case "收入":
		return 1
	case "转账":
		return 2
	default:
		return 3
	}
}

// formatDiagnostics 把诊断渲染为可读文本，无可诊断时返回空串。
func formatDiagnostics(diags []Diagnostic) string {
	if len(diags) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "\n=== 异常行 (%d) ===\n", len(diags))
	for _, d := range diags {
		fmt.Fprintf(&b, "[%s] 行 %d: %s\n", d.Source, d.Row, d.Reason)
	}
	return b.String()
}
