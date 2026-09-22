package service

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestNormalizeBinanceEventsDeduplicatesAndMergesOrderFills(t *testing.T) {
	events := []BinanceEvent{
		{EventID: "trade-1", OrderID: "order-1", EventType: "spot_trade", Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", Side: "buy", Quantity: decimal.RequireFromString("0.01"), QuoteQuantity: decimal.RequireFromString("650"), Time: time.Unix(2, 0)},
		{EventID: "trade-1", OrderID: "order-1", EventType: "spot_trade", Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", Side: "buy", Quantity: decimal.RequireFromString("0.01"), QuoteQuantity: decimal.RequireFromString("650"), Time: time.Unix(2, 0)},
		{EventID: "trade-2", OrderID: "order-1", EventType: "spot_trade", Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", Side: "buy", Quantity: decimal.RequireFromString("0.02"), QuoteQuantity: decimal.RequireFromString("1300"), Time: time.Unix(3, 0)},
	}
	got, diagnostics := NormalizeBinanceEvents(events)
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
	if len(got) != 1 {
		t.Fatalf("期望一个合并订单，实际 %d: %+v", len(got), got)
	}
	if !got[0].Quantity.Equal(decimal.RequireFromString("0.03")) || !got[0].QuoteQuantity.Equal(decimal.RequireFromString("1950")) {
		t.Fatalf("合并金额错误: %+v", got[0])
	}
}
