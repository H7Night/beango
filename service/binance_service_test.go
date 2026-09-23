package service

import "testing"

func TestRunBinanceSpotFixtureDryRun(t *testing.T) {
	path := findZipByHeader(t, "交易对", "订单号")
	if err := RunBinanceCLI(BinanceCLIOptions{Input: path, DryRun: true}); err != nil {
		t.Fatalf("Spot dry-run 失败: %v", err)
	}
}

func TestRunBinanceFuturesFixtureDryRunReportsUnsupported(t *testing.T) {
	path := findZipByHeader(t, "用户ID", "执行金额")
	if err := RunBinanceCLI(BinanceCLIOptions{Input: path, DryRun: true}); err != nil {
		t.Fatalf("Futures dry-run 失败: %v", err)
	}
}
