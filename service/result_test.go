package service

import (
	"strings"
	"testing"
)

func TestFormatDiagnostics(t *testing.T) {
	if got := formatDiagnostics(nil); got != "" {
		t.Fatalf("空诊断应返回空串，实际 %q", got)
	}
	diags := []Diagnostic{
		{Source: "alipay", Row: 3, Reason: `金额无法解析: "abc"`},
		{Source: "wechat", Row: 5, Reason: `金额无法解析: "x"`},
	}
	got := formatDiagnostics(diags)
	for _, want := range []string{"alipay", "行 3", `金额无法解析: "abc"`, "wechat", "行 5"} {
		if !strings.Contains(got, want) {
			t.Errorf("输出缺少 %q:\n%s", want, got)
		}
	}
}
