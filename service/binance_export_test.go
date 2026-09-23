package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestRenderBinanceTransactionIncludesMetadataAndPostings(t *testing.T) {
	txn := GeneratedTransaction{
		Date: time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local), Time: "12:34:56", Payee: "Binance", Narration: "spot_trade",
		Metadata: map[string]string{"bill": "binance", "source_id": "trade-1", "order_id": "order-1"},
		Postings: []GeneratedPosting{
			{Account: "Assets:Binance:BTC", Currency: "BTC", Units: decimal.RequireFromString("0.01"), Cost: decimal.RequireFromString("65000"), CostCurrency: "USDT", HasCost: true},
			{Account: "Assets:Binance:USDT", Currency: "USDT", Units: decimal.RequireFromString("-650")},
		},
	}
	got := RenderBinanceTransaction(txn)
	for _, want := range []string{`bill: "binance"`, `source_id: "trade-1"`, `order_id: "order-1"`, "Assets:Binance:BTC", "Assets:Binance:USDT", "{65000 USDT}"} {
		if !strings.Contains(got, want) {
			t.Errorf("输出缺少 %q:\n%s", want, got)
		}
	}
}

func TestWriteBinanceTransactionsCreatesCryptoMonthAndDeclarations(t *testing.T) {
	base := t.TempDir()
	txns := []GeneratedTransaction{
		{Date: time.Date(2026, 9, 2, 0, 0, 0, 0, time.Local), Time: "12:00:00", Payee: "Binance", Narration: "buy", Metadata: map[string]string{"bill": "binance", "source_id": "2"}, Postings: []GeneratedPosting{{Account: "Assets:Binance:BTC", Currency: "BTC", Units: decimal.RequireFromString("0.01")}}},
		{Date: time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local), Time: "12:00:00", Payee: "Binance", Narration: "buy", Metadata: map[string]string{"bill": "binance", "source_id": "1"}, Postings: []GeneratedPosting{{Account: "Assets:Binance:USDT", Currency: "USDT", Units: decimal.RequireFromString("-650")}}},
	}
	if err := WriteBinanceTransactions(txns, base, "2-crypto"); err != nil {
		t.Fatal(err)
	}
	monthFile := filepath.Join(base, "2026", "2-crypto", "09.bean")
	content, err := os.ReadFile(monthFile)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if strings.Index(text, `source_id: "1"`) > strings.Index(text, `source_id: "2"`) {
		t.Fatal("交易未按时间正序输出")
	}
	declarations, err := os.ReadFile(filepath.Join(base, "2026", "2-crypto", "00.bean"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(declarations), "open Assets:Binance:") != 2 {
		t.Fatalf("账户声明重复或缺失:\n%s", declarations)
	}
}

func TestWriteBinanceTransactionsMergesExistingMonth(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "2026", "2-crypto")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	old := "2026-09-01 * \"Binance\" \"old\"\n    source_id: \"old\"\n    Assets:Binance:BTC  1 BTC\n\n"
	if err := os.WriteFile(filepath.Join(dir, "09.bean"), []byte(old), 0644); err != nil {
		t.Fatal(err)
	}
	txn := GeneratedTransaction{Date: time.Date(2026, 9, 2, 0, 0, 0, 0, time.Local), Time: "12:00:00", Payee: "Binance", Narration: "new", Metadata: map[string]string{"source_id": "new"}, Postings: []GeneratedPosting{{Account: "Assets:Binance:BTC", Currency: "BTC", Units: decimal.RequireFromString("2")}}}
	if err := WriteBinanceTransactions([]GeneratedTransaction{txn}, base, "2-crypto"); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "09.bean"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `source_id: "old"`) || !strings.Contains(string(content), `source_id: "new"`) {
		t.Fatalf("旧交易被覆盖: %s", content)
	}
}
