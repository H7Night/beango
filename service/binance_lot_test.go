package service

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func applyBuy(t *testing.T, book *LotBook, asset, quantity, cost, currency string) {
	t.Helper()
	_, diagnostics, err := ApplyFIFO(book, BinanceEvent{
		EventID: "buy-" + quantity + "-" + cost, EventType: "spot_trade", Side: "buy",
		BaseAsset: asset, QuoteAsset: currency,
		Quantity: decimal.RequireFromString(quantity), QuoteQuantity: decimal.RequireFromString(cost),
		Price: decimal.RequireFromString(cost).Div(decimal.RequireFromString(quantity)), Time: time.Unix(0, 0),
	})
	if err != nil || len(diagnostics) != 0 {
		t.Fatal(err, diagnostics)
	}
}

func sellEvent(symbol, quantity, price, proceeds string) BinanceEvent {
	base, quote, _ := SplitBinanceSymbol(symbol)
	return BinanceEvent{EventID: "sell-" + quantity + "-" + price, EventType: "spot_trade", Side: "sell",
		Symbol: symbol, BaseAsset: base, QuoteAsset: quote,
		Quantity: decimal.RequireFromString(quantity), Price: decimal.RequireFromString(price),
		QuoteQuantity: decimal.RequireFromString(proceeds), Time: time.Unix(1, 0)}
}

func findPosting(t *testing.T, postings []GeneratedPosting, account string) GeneratedPosting {
	t.Helper()
	for _, posting := range postings {
		if posting.Account == account {
			return posting
		}
	}
	t.Fatalf("找不到 posting=%s", account)
	return GeneratedPosting{}
}

func TestApplyFIFOPartialSell(t *testing.T) {
	book := NewLotBook()
	applyBuy(t, book, "BTC", "0.01", "6.5", "USDT")
	applyBuy(t, book, "BTC", "0.02", "14", "USDT")
	txn, diagnostics, err := ApplyFIFO(book, sellEvent("BTCUSDT", "0.015", "800", "12"))
	if err != nil || len(diagnostics) != 0 {
		t.Fatal(err, diagnostics)
	}
	gain := findPosting(t, txn.Postings, "Income:CapitalGains:Crypto")
	if !gain.Units.Equal(decimal.NewFromInt(-2)) {
		t.Fatalf("FIFO gain=%s", gain.Units)
	}
	if len(book.Accounts["BTC"]) != 1 || !book.Accounts["BTC"][0].Quantity.Equal(decimal.RequireFromString("0.015")) {
		t.Fatalf("剩余批次=%+v", book.Accounts["BTC"])
	}
}

func TestApplyFIFORejectsOversellWithoutCreatingTransaction(t *testing.T) {
	book := NewLotBook()
	applyBuy(t, book, "BTC", "0.01", "650", "USDT")
	txn, diagnostics, err := ApplyFIFO(book, sellEvent("BTCUSDT", "0.02", "800", "16"))
	if err == nil && len(diagnostics) == 0 {
		t.Fatal("超出库存应返回错误或诊断")
	}
	if len(txn.Postings) != 0 {
		t.Fatal("失败卖出不应生成分录")
	}
}

func TestApplyFIFODeduplicatesEventID(t *testing.T) {
	book := NewLotBook()
	event := BinanceEvent{EventID: "buy-1", EventType: "spot_trade", Side: "buy", BaseAsset: "BTC", QuoteAsset: "USDT", Quantity: decimal.RequireFromString("0.01"), QuoteQuantity: decimal.RequireFromString("650"), Price: decimal.RequireFromString("65000"), Time: time.Unix(0, 0)}
	if _, diagnostics, err := ApplyFIFO(book, event); err != nil || len(diagnostics) != 0 {
		t.Fatal(err, diagnostics)
	}
	txn, diagnostics, err := ApplyFIFO(book, event)
	if err != nil || len(diagnostics) != 0 || len(txn.Postings) != 0 {
		t.Fatalf("重复事件未幂等: %+v %v %v", txn, diagnostics, err)
	}
	if !book.Accounts["BTC"][0].Quantity.Equal(decimal.RequireFromString("0.01")) {
		t.Fatal("重复事件增加了库存")
	}
}
