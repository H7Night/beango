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
	Account       string
	Currency      string
	Units         decimal.Decimal
	Cost          decimal.Decimal
	CostCurrency  string
	Price         decimal.Decimal
	PriceCurrency string
	HasCost       bool
	HasPrice      bool
}

type GeneratedTransaction struct {
	Date      time.Time
	Time      string
	Payee     string
	Narration string
	Metadata  map[string]string
	Postings  []GeneratedPosting
}

// cashLikeAssets 是按「现金类」处理的稳定币：不建成本批次，用 @ 价格表示兑换。
var cashLikeAssets = map[string]bool{
	"USDT": true, "USDC": true, "BUSD": true, "FDUSD": true, "TUSD": true,
	"USDP": true, "DAI": true, "USD": true,
}

func isCashLikeAsset(asset string) bool { return cashLikeAssets[strings.ToUpper(asset)] }

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
		return applyBuyFIFO(book, event, txn)
	}
	if event.EventType == "spot_trade" && event.Side == "sell" {
		return applySell(book, event, txn)
	}
	if event.EventType == "withdrawal" {
		return applyWithdrawalFIFO(book, event, txn)
	}
	return GeneratedTransaction{}, []BinanceDiagnostic{{Source: event.Source, Reason: "未知或暂不支持的 Binance 事件类型", Raw: rawEvent(event)}}, nil
}

func applyBuyFIFO(book *LotBook, event BinanceEvent, txn GeneratedTransaction) (GeneratedTransaction, []BinanceDiagnostic, error) {
	if event.Quantity.IsZero() || event.QuoteQuantity.IsNegative() || event.QuoteQuantity.IsZero() {
		return GeneratedTransaction{}, nil, fmt.Errorf("买入事件数量或金额无效: %s", event.EventID)
	}
	quoteAccount := cryptoAccount(event.QuoteAsset)
	if event.EventType == "fiat_buy" && event.QuoteAsset == "CNY" {
		quoteAccount = "Assets:Crypto"
	}
	if isCashLikeAsset(event.BaseAsset) {
		txn.Postings = append(txn.Postings, GeneratedPosting{Account: cryptoAccount(event.BaseAsset), Currency: event.BaseAsset, Units: event.Quantity, Price: event.Price, PriceCurrency: event.QuoteAsset, HasPrice: true})
		txn.Postings = append(txn.Postings, GeneratedPosting{Account: quoteAccount, Currency: event.QuoteAsset, Units: event.QuoteQuantity.Neg()})
		if event.FeeAsset != "" && !event.Fee.IsZero() {
			appendFeePostings(&txn, event)
		}
		return txn, nil, nil
	}
	unitCost := event.QuoteQuantity.Div(event.Quantity)
	netQuantity := event.Quantity
	if event.FeeAsset == event.BaseAsset && !event.Fee.IsZero() {
		netQuantity = netQuantity.Sub(event.Fee)
		txn.Postings = append(txn.Postings, GeneratedPosting{Account: "Expenses:Crypto:Fees:Trading", Currency: event.BaseAsset, Units: event.Fee, Cost: unitCost, CostCurrency: event.QuoteAsset, HasCost: true})
	}
	book.Accounts[event.BaseAsset] = append(book.Accounts[event.BaseAsset], Lot{Quantity: netQuantity, Cost: unitCost, CostCurrency: event.QuoteAsset, AcquiredAt: event.Time, SourceID: event.EventID})
	txn.Postings = append(txn.Postings, GeneratedPosting{Account: cryptoAccount(event.BaseAsset), Currency: event.BaseAsset, Units: netQuantity, Cost: unitCost, CostCurrency: event.QuoteAsset, HasCost: true})
	txn.Postings = append(txn.Postings, GeneratedPosting{Account: quoteAccount, Currency: event.QuoteAsset, Units: event.QuoteQuantity.Neg()})
	if event.FeeAsset != "" && event.FeeAsset != event.BaseAsset {
		appendFeePostings(&txn, event)
	}
	return txn, nil, nil
}

func applySell(book *LotBook, event BinanceEvent, txn GeneratedTransaction) (GeneratedTransaction, []BinanceDiagnostic, error) {
	if isCashLikeAsset(event.BaseAsset) {
		txn.Postings = append(txn.Postings, GeneratedPosting{Account: cryptoAccount(event.BaseAsset), Currency: event.BaseAsset, Units: event.Quantity.Neg(), Price: event.Price, PriceCurrency: event.QuoteAsset, HasPrice: true})
		txn.Postings = append(txn.Postings, GeneratedPosting{Account: cryptoAccount(event.QuoteAsset), Currency: event.QuoteAsset, Units: event.QuoteQuantity})
		if event.FeeAsset != "" && !event.Fee.IsZero() {
			appendFeePostings(&txn, event)
		}
		return txn, nil, nil
	}
	return applySellFIFO(book, event, txn)
}

func applySellFIFO(book *LotBook, event BinanceEvent, txn GeneratedTransaction) (GeneratedTransaction, []BinanceDiagnostic, error) {
	var matching, others []Lot
	for _, lot := range book.Accounts[event.BaseAsset] {
		if lot.CostCurrency == event.QuoteAsset {
			matching = append(matching, lot)
		} else {
			others = append(others, lot)
		}
	}
	feeQuantity := decimal.Zero
	if event.FeeAsset == event.BaseAsset {
		feeQuantity = event.Fee
	}
	required := event.Quantity.Add(feeQuantity)
	available := decimal.Zero
	for _, lot := range matching {
		available = available.Add(lot.Quantity)
	}
	if available.LessThan(required) {
		return GeneratedTransaction{}, []BinanceDiagnostic{{Source: event.Source, Reason: "FIFO 库存不足（成本货币 " + event.QuoteAsset + "）", Raw: rawEvent(event)}}, nil
	}

	remaining := required
	saleRemaining := event.Quantity
	costQuantity := decimal.Zero
	costFee := decimal.Zero
	var kept []Lot
	for _, lot := range matching {
		if remaining.IsZero() {
			kept = append(kept, lot)
			continue
		}
		used := lot.Quantity
		if used.GreaterThan(remaining) {
			used = remaining
		}
		txn.Postings = append(txn.Postings, GeneratedPosting{Account: cryptoAccount(event.BaseAsset), Currency: event.BaseAsset, Units: used.Neg(), Cost: lot.Cost, CostCurrency: lot.CostCurrency, Price: event.Price, PriceCurrency: event.QuoteAsset, HasCost: true, HasPrice: true})
		saleUsed := used
		if saleUsed.GreaterThan(saleRemaining) {
			saleUsed = saleRemaining
		}
		costQuantity = costQuantity.Add(saleUsed.Mul(lot.Cost))
		costFee = costFee.Add(used.Sub(saleUsed).Mul(lot.Cost))
		saleRemaining = saleRemaining.Sub(saleUsed)
		remaining = remaining.Sub(used)
		left := lot.Quantity.Sub(used)
		if left.GreaterThan(decimal.Zero) {
			kept = append(kept, Lot{Quantity: left, Cost: lot.Cost, CostCurrency: lot.CostCurrency, AcquiredAt: lot.AcquiredAt, SourceID: lot.SourceID})
		}
	}
	book.Accounts[event.BaseAsset] = append(others, kept...)

	proceeds := event.QuoteQuantity
	txn.Postings = append(txn.Postings, GeneratedPosting{Account: cryptoAccount(event.QuoteAsset), Currency: event.QuoteAsset, Units: proceeds})
	gain := proceeds.Sub(costQuantity)
	if gain.GreaterThan(decimal.Zero) {
		txn.Postings = append(txn.Postings, GeneratedPosting{Account: "Income:CapitalGains:Crypto", Currency: event.QuoteAsset, Units: gain.Neg()})
	} else if gain.LessThan(decimal.Zero) {
		txn.Postings = append(txn.Postings, GeneratedPosting{Account: "Expenses:CapitalLoss:Crypto", Currency: event.QuoteAsset, Units: gain.Abs()})
	}
	if !feeQuantity.IsZero() {
		txn.Postings = append(txn.Postings, GeneratedPosting{Account: "Expenses:Crypto:Fees:Trading", Currency: event.BaseAsset, Units: feeQuantity, Cost: costFee.Div(feeQuantity), CostCurrency: event.QuoteAsset, HasCost: true})
	} else if event.FeeAsset != "" && !event.Fee.IsZero() {
		appendFeePostings(&txn, event)
	}
	return txn, nil, nil
}

func applyWithdrawalFIFO(book *LotBook, event BinanceEvent, txn GeneratedTransaction) (GeneratedTransaction, []BinanceDiagnostic, error) {
	lots := book.Accounts[event.BaseAsset]
	remaining := event.Quantity.Add(event.Fee)
	available := decimal.Zero
	for _, lot := range lots {
		available = available.Add(lot.Quantity)
	}
	if available.LessThan(remaining) {
		return GeneratedTransaction{}, []BinanceDiagnostic{{Source: event.Source, Reason: "提现数量和手续费超过 FIFO 库存", Raw: rawEvent(event)}}, nil
	}
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
		txn.Postings = append(txn.Postings, GeneratedPosting{Account: cryptoAccount(event.BaseAsset), Currency: event.BaseAsset, Units: used.Neg(), Cost: lot.Cost, CostCurrency: lot.CostCurrency, HasCost: true})
		remaining = remaining.Sub(used)
		left := lot.Quantity.Sub(used)
		if left.GreaterThan(decimal.Zero) {
			kept = append(kept, Lot{Quantity: left, Cost: lot.Cost, CostCurrency: lot.CostCurrency, AcquiredAt: lot.AcquiredAt, SourceID: lot.SourceID})
		}
	}
	book.Accounts[event.BaseAsset] = kept
	if !event.Fee.IsZero() {
		txn.Postings = append(txn.Postings, GeneratedPosting{Account: "Expenses:Crypto:Fees:Withdrawal", Currency: event.FeeAsset, Units: event.Fee})
	}
	return txn, nil, nil
}

func appendFeePostings(txn *GeneratedTransaction, event BinanceEvent) {
	if event.Fee.IsZero() || event.FeeAsset == "" {
		return
	}
	txn.Postings = append(txn.Postings, GeneratedPosting{Account: "Expenses:Crypto:Fees:Trading", Currency: event.FeeAsset, Units: event.Fee})
	if event.FeeAsset == event.QuoteAsset {
		txn.Postings = append(txn.Postings, GeneratedPosting{Account: cryptoAccount(event.FeeAsset), Currency: event.FeeAsset, Units: event.Fee.Neg()})
	}
}

func cryptoAccount(asset string) string { return "Assets:Binance:" + strings.ToUpper(asset) }

func rawEvent(event BinanceEvent) []string {
	return []string{event.Source, event.EventID, event.EventType, event.Symbol, event.BaseAsset, event.QuoteAsset}
}
