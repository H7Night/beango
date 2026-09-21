package service

import (
	"strings"
	"testing"
)

func wechatHeaderRow() []string {
	return []string{"交易时间", "交易类型", "交易对方", "商品", "收/支", "金额(元)", "支付方式", "当前状态", "交易单号", "商户单号", "备注"}
}

func wechatRow(uuid, amount string) []string {
	return []string{
		"2026-09-10 19:13:06", "商户消费", "购票支付", "购票支付", "支出",
		amount, "招商银行信用卡(2035)", "支付成功", uuid, "6133202691000425524", "/",
	}
}

func TestTransWechatInvalidAmountSkipped(t *testing.T) {
	chdirRepoRoot(t)

	records := [][]string{
		wechatHeaderRow(),
		wechatRow("uuid-good", "17"),
		wechatRow("uuid-bad", "abc"),
	}
	res, err := TransWechat(records, false)
	if err != nil {
		t.Fatalf("TransWechat 失败: %v", err)
	}
	if len(res.Entries) != 1 {
		t.Fatalf("期望 1 条有效条目，实际 %d", len(res.Entries))
	}
	if len(res.Diagnostics) != 1 {
		t.Fatalf("期望 1 条诊断，实际 %d", len(res.Diagnostics))
	}
	if res.Diagnostics[0].Source != "wechat" {
		t.Errorf("Source 应为 wechat，实际 %q", res.Diagnostics[0].Source)
	}
	if !strings.Contains(res.Diagnostics[0].Reason, "abc") {
		t.Errorf("诊断原因应含原值，实际 %q", res.Diagnostics[0].Reason)
	}
}

func TestTransWechatRejectsMissingHeader(t *testing.T) {
	chdirRepoRoot(t)
	records := [][]string{{"foo", "bar"}, {"a", "b"}}
	_, err := TransWechat(records, false)
	if err == nil || !strings.Contains(err.Error(), "导入文件不符合微信格式") {
		t.Fatalf("期望微信格式错误，实际 %v", err)
	}
}

func TestTransWechatRejectsShortInput(t *testing.T) {
	chdirRepoRoot(t)
	if _, err := TransWechat([][]string{wechatHeaderRow()}, false); err == nil {
		t.Fatal("仅表头应报错")
	}
}

func TestTransWechatAcceptsHeader(t *testing.T) {
	chdirRepoRoot(t)
	res, err := TransWechat([][]string{wechatHeaderRow(), wechatRow("u1", "17")}, false)
	if err != nil {
		t.Fatalf("TransWechat 失败: %v", err)
	}
	if len(res.Entries) != 1 {
		t.Fatalf("期望 1 条，实际 %d", len(res.Entries))
	}
}
