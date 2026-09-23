package service

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

func TestRunBinanceSyncStopsOnRequiredAPIFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v3/myTrades" {
			_, _ = io.WriteString(w, `[{"id":1,"orderId":1,"symbol":"BTCUSDT","price":"65000","qty":"0.01","quoteQty":"650","commission":"0","commissionAsset":"USDT","time":1760000000000,"isBuyer":true}]`)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"code":-1000,"msg":"failure"}`)
	}))
	defer server.Close()
	t.Setenv("BINANCE_API_KEY", "key")
	t.Setenv("BINANCE_API_SECRET", "secret")
	t.Setenv("BINANCE_BASE_URL", server.URL)
	if err := RunBinanceCLI(BinanceCLIOptions{Sync: true, DryRun: true, Symbols: []string{"BTCUSDT"}}); err == nil {
		t.Fatal("必要 API 请求失败时不应继续导出部分结果")
	}
}
