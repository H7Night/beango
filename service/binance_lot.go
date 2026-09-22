package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type Lot struct {
	Quantity     decimal.Decimal
	Cost         decimal.Decimal // 每单位成本
	CostCurrency string
	AcquiredAt   time.Time
	SourceID     string
}

type LotBook struct {
	Accounts  map[string][]Lot
	Processed map[string]bool
}

type GeneratedPosting struct {
	Account  string
	Currency string
	Units    decimal.Decimal
	Cost     decimal.Decimal
	Price    decimal.Decimal
	HasCost  bool
	HasPrice bool
}

type GeneratedTransaction struct {
	Date      time.Time
	Time      string
	Payee     string
	Narration string
	Metadata  map[string]string
	Postings  []GeneratedPosting
}

func NewLotBook() *LotBook {
	return &LotBook{Accounts: make(map[string][]Lot), Processed: make(map[string]bool)}
}

func ApplyFIFO(book *LotBook, event BinanceEvent) (GeneratedTransaction, []BinanceDiagnostic, error) {
	if book == nil {
		return GeneratedTransaction{}, nil, fmt.Errorf("lot book 为空")
	}
	if event.EventID != "" && book.Processed[event.EventID] {
		return GeneratedTransaction{}, nil, nil
	}
	if event.EventID != "" {
		defer func() { book.Processed[event.EventID] = true }()
	}

	txn := GeneratedTransaction{
		Date: event.Time, Time: event.Time.Format("15:04:05"),
		Payee: "Binance", Narration: event.EventType,
		Metadata: map[string]string{"bill": "binance", "source_id": event.EventID, "order_id": event.OrderID},
	}
	if event.EventType == "deposit" {
		return GeneratedTransaction{}, []BinanceDiagnostic{{Source: event.Source, Reason: "外部充值缺少成本基础，未自动建批次", Raw: rawEvent(event)}}, nil
	}
	if event.EventType == "fiat_buy" || (event.EventType == "spot_trade" && event.Side == "buy") {
		if event.Quantity.IsZero() || event.QuoteQuantity.IsNegative() || event.QuoteQuantity.IsZero() {
			return GeneratedTransaction{}, nil, fmt.Errorf("买入事件数量或金额无效: %s", event.EventID)
		}
		cost := event.QuoteQuantity.Div(event.Quantity)
		book.Accounts[event.BaseAsset] = append(book.Accounts[event.BaseAsset], Lot{Quantity: event.Quantity, Cost: cost, CostCurrency: event.QuoteAsset, AcquiredAt: event.Time, SourceID: event.EventID})
		baseUnits := event.Quantity
		if event.FeeAsset == event.BaseAsset && !event.Fee.IsZero() {
			baseUnits = baseUnits.Sub(event.Fee)
			txn.Postings = append(txn.Postings, GeneratedPosting{Account: cryptoAccount(event.BaseAsset), Currency: event.BaseAsset, Units: event.Fee, Cost: cost, HasCost: true})
		}
		txn.Postings = append(txn.Postings, GeneratedPosting{Account: cryptoAccount(event.BaseAsset), Currency: event.BaseAsset, Units: baseUnits, Cost: cost, HasCost: true})
		quoteAccount := cryptoAccount(event.QuoteAsset)
		if event.EventType == "fiat_buy" && event.QuoteAsset == "CNY" {
			quoteAccount = "Assets:Crypto"
		}
		txn.Postings = append(txn.Postings, GeneratedPosting{Account: quoteAccount, Currency: event.QuoteAsset, Units: event.QuoteQuantity.Neg()})
		if event.FeeAsset != "" && event.FeeAsset != event.BaseAsset {
			txn.Postings = appendFeePostings(&txn, event)
		}
		return txn, nil, nil
	}
	if event.EventType == "spot_trade" && event.Side == "sell" {
		return applySellFIFO(book, event, txn)
	}
	if event.EventType == "withdrawal" {
		return applyWithdrawalFIFO(book, event, txn)
	}
	return GeneratedTransaction{}, []BinanceDiagnostic{{Source: event.Source, Reason: "未知或暂不支持的 Binance 事件类型", Raw: rawEvent(event)}}, nil
}

func applySellFIFO(book *LotBook, event BinanceEvent, txn GeneratedTransaction) (GeneratedTransaction, []BinanceDiagnostic, error) {
	lots := book.Accounts[event.BaseAsset]
	remaining := event.Quantity
	totalCost := decimal.Zero
	for _, lot := range lots {
		if remaining.GreaterThan(lot.Quantity) {
			totalCost = totalCost.Add(lot.Quantity.Mul(lot.Cost))
			remaining = remaining.Sub(lot.Quantity)
		} else {
			totalCost = totalCost.Add(remaining.Mul(lot.Cost))
			remaining = decimal.Zero
			break
		}
	}
	if remaining.GreaterThan(decimal.Zero) {
		return GeneratedTransaction{}, []BinanceDiagnostic{{Source: event.Source, Reason: "FIFO 库存不足，无法安全计算卖出成本", Raw: rawEvent(event)}}, nil
	}
	remaining = event.Quantity
	var kept []Lot
	for _, lot := range lots {
		if remaining.IsZero() {
			kept = append(kept, lot)
			continue
		}
		used := lot.Quantity
		if used.GreaterThan(remaining) {
			used = remaining
		}
		txn.Postings = append(txn.Postings, GeneratedPosting{Account: cryptoAccount(event.BaseAsset), Currency: event.BaseAsset, Units: used.Neg(), Cost: lot.Cost, Price: event.Price, HasCost: true, HasPrice: true})
		remaining = remaining.Sub(used)
		left := lot.Quantity.Sub(used)
		if left.GreaterThan(decimal.Zero) {
			kept = append(kept, Lot{Quantity: left, Cost: lot.Cost, CostCurrency: lot.CostCurrency, AcquiredAt: lot.AcquiredAt, SourceID: lot.SourceID})
		}
	}
	book.Accounts[event.BaseAsset] = kept
	proceeds := event.QuoteQuantity
	txn.Postings = append(txn.Postings, GeneratedPosting{Account: cryptoAccount(event.QuoteAsset), Currency: event.QuoteAsset, Units: proceeds})
	gain := proceeds.Sub(totalCost)
	if gain.GreaterThan(decimal.Zero) {
		txn.Postings = append(txn.Postings, GeneratedPosting{Account: "Income:CapitalGains:Crypto", Currency: event.QuoteAsset, Units: gain.Neg()})
	} else if gain.LessThan(decimal.Zero) {
		txn.Postings = append(txn.Postings, GeneratedPosting{Account: "Expenses:CapitalLoss:Crypto", Currency: event.QuoteAsset, Units: gain.Abs()})
	}
	if !event.Fee.IsZero() {
		txn.Postings = appendFeePostings(&txn, event)
	}
	return txn, nil, nil
}

func applyWithdrawalFIFO(book *LotBook, event BinanceEvent, txn GeneratedTransaction) (GeneratedTransaction, []BinanceDiagnostic, error) {
	lots := book.Accounts[event.BaseAsset]
	remaining := event.Quantity.Add(event.Fee)
	var kept []Lot
	for _, lot := range lots {
		if remaining.IsZero() {
			kept = append(kept, lot)
			continue
		}
		used := lot.Quantity
		if used.GreaterThan(remaining) {
			used = remaining
		}
		txn.Postings = append(txn.Postings, GeneratedPosting{Account: cryptoAccount(event.BaseAsset), Currency: event.BaseAsset, Units: used.Neg(), Cost: lot.Cost, HasCost: true})
		remaining = remaining.Sub(used)
		left := lot.Quantity.Sub(used)
		if left.GreaterThan(decimal.Zero) {
			kept = append(kept, Lot{Quantity: left, Cost: lot.Cost, CostCurrency: lot.CostCurrency, AcquiredAt: lot.AcquiredAt, SourceID: lot.SourceID})
		}
	}
	if remaining.GreaterThan(decimal.Zero) {
		return GeneratedTransaction{}, []BinanceDiagnostic{{Source: event.Source, Reason: "提现数量和手续费超过 FIFO 库存", Raw: rawEvent(event)}}, nil
	}
	book.Accounts[event.BaseAsset] = kept
	if !event.Fee.IsZero() {
		txn.Postings = append(txn.Postings, GeneratedPosting{Account: "Expenses:Crypto:Fees:Withdrawal", Currency: event.FeeAsset, Units: event.Fee})
	}
	return txn, nil, nil
}

func appendFeePostings(txn *GeneratedTransaction, event BinanceEvent) []GeneratedPosting {
	if event.Fee.IsZero() || event.FeeAsset == "" {
		return nil
	}
	postings := []GeneratedPosting{{Account: "Expenses:Crypto:Fees:Trading", Currency: event.FeeAsset, Units: event.Fee}}
	if event.FeeAsset == event.QuoteAsset {
		postings = append(postings, GeneratedPosting{Account: cryptoAccount(event.FeeAsset), Currency: event.FeeAsset, Units: event.Fee.Neg()})
	}
	txn.Postings = append(txn.Postings, postings...)
	return postings
}

func cryptoAccount(asset string) string { return "Assets:Binance:" + strings.ToUpper(asset) }

func rawEvent(event BinanceEvent) []string {
	return []string{event.Source, event.EventID, event.EventType, event.Symbol, event.BaseAsset, event.QuoteAsset}
}
