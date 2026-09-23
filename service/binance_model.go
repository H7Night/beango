package service

import (
	"time"

	"github.com/shopspring/decimal"
)

// BinanceEvent 是 API/CSV 归一化后的 Binance 账户事件。
type BinanceEvent struct {
	Source        string
	EventID       string
	OrderID       string
	EventType     string
	Symbol        string
	BaseAsset     string
	QuoteAsset    string
	Side          string
	Time          time.Time
	Quantity      decimal.Decimal
	QuoteQuantity decimal.Decimal
	Price         decimal.Decimal
	Fee           decimal.Decimal
	FeeAsset      string
	Raw           map[string]string
}

// BinanceDiagnostic 记录无法安全进入账本的 Binance 原始记录。
type BinanceDiagnostic struct {
	Source string
	Reason string
	Row    int
	Raw    []string
}

// BinanceParseResult 是单个 API/CSV 输入的解析结果。
type BinanceParseResult struct {
	Events      []BinanceEvent
	Diagnostics []BinanceDiagnostic
}
