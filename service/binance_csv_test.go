package service

import (
	"archive/zip"
	"encoding/csv"
	"path/filepath"
	"strings"
	"testing"
)

func findZipByHeader(t *testing.T, required ...string) string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("..", "test", "Binance-*.zip"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("找不到 Binance fixture: %v", err)
	}
	for _, path := range paths {
		archive, err := zip.OpenReader(path)
		if err != nil {
			continue
		}
		for _, file := range archive.File {
			reader, err := file.Open()
			if err != nil {
				continue
			}
			header, err := csv.NewReader(reader).Read()
			_ = reader.Close()
			if err != nil {
				continue
			}
			joined := strings.Join(header, ",")
			matched := true
			for _, item := range required {
				if !strings.Contains(joined, item) {
					matched = false
					break
				}
			}
			if matched {
				_ = archive.Close()
				return path
			}
		}
		_ = archive.Close()
	}
	t.Fatalf("找不到 header=%v 的 fixture", required)
	return ""
}

func findEvent(t *testing.T, events []BinanceEvent, id string) BinanceEvent {
	t.Helper()
	for _, event := range events {
		if event.EventID == id {
			return event
		}
	}
	t.Fatalf("找不到 event id=%s", id)
	return BinanceEvent{}
}

func assertEventHas(t *testing.T, events []BinanceEvent, symbol, base, quote string) {
	t.Helper()
	for _, event := range events {
		if event.Symbol == symbol && event.BaseAsset == base && event.QuoteAsset == quote {
			return
		}
	}
	t.Fatalf("找不到 %s/%s/%s", symbol, base, quote)
}

func parseKnownSpotFixture(t *testing.T) BinanceParseResult {
	t.Helper()
	path := findZipByHeader(t, "交易对", "订单号")
	result, err := ParseBinanceZip(path)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func hasDiagnosticContaining(items []BinanceDiagnostic, text string) bool {
	for _, item := range items {
		if strings.Contains(item.Reason, text) {
			return true
		}
	}
	return false
}

func TestParseSpotFixture(t *testing.T) {
	path := findZipByHeader(t, "交易对", "订单号")
	result, err := ParseBinanceZip(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Events) == 0 {
		t.Fatal("Spot fixture 应产生事件")
	}
	if result.Events[0].EventType != "spot_trade" {
		t.Fatalf("事件类型=%s", result.Events[0].EventType)
	}
	assertEventHas(t, result.Events, "BTCUSDT", "BTC", "USDT")
	assertEventHas(t, result.Events, "USDCUSDT", "USDC", "USDT")
	assertEventHas(t, result.Events, "ETHUSDC", "ETH", "USDC")
	assertEventHas(t, result.Events, "USDTUSD", "USDT", "USD")
}

func TestParseSpotUnitsAndQuantities(t *testing.T) {
	result := parseKnownSpotFixture(t)
	trade := findEvent(t, result.Events, "1673595238")
	if trade.BaseAsset != "USDC" || trade.QuoteAsset != "USDT" {
		t.Fatal("USDCUSDT 拆分错误")
	}
	if trade.Quantity.String() != "649" {
		t.Fatalf("数量=%s", trade.Quantity)
	}
	if trade.QuoteQuantity.String() != "649.07788" {
		t.Fatalf("报价金额=%s", trade.QuoteQuantity)
	}
}

func TestParseFuturesFixtureIsUnsupported(t *testing.T) {
	path := findZipByHeader(t, "用户ID", "执行金额")
	result, err := ParseBinanceZip(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Events) != 0 {
		t.Fatal("第一阶段不应生成 Futures 现货事件")
	}
	if !hasDiagnosticContaining(result.Diagnostics, "Futures") {
		t.Fatal("应明确报告 Futures 暂不支持")
	}
}
