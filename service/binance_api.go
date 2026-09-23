package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// BinanceClient 是 Binance signed USER_DATA API 的只读客户端。
type BinanceClient struct {
	BaseURL        string
	APIKey         string
	APISecret      string
	HTTPClient     *http.Client
	TradePageSize  int
	WalletPageSize int
}

func NewBinanceClientFromEnv() (*BinanceClient, error) {
	key := strings.TrimSpace(os.Getenv("BINANCE_API_KEY"))
	secret := strings.TrimSpace(os.Getenv("BINANCE_API_SECRET"))
	if key == "" || secret == "" {
		return nil, fmt.Errorf("缺少 BINANCE_API_KEY 或 BINANCE_API_SECRET")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("BINANCE_BASE_URL")), "/")
	if baseURL == "" {
		baseURL = "https://api.binance.com"
	}
	return &BinanceClient{
		BaseURL:        baseURL,
		APIKey:         key,
		APISecret:      secret,
		HTTPClient:     &http.Client{Timeout: 30 * time.Second},
		TradePageSize:  1000,
		WalletPageSize: 1000,
	}, nil
}

func (c *BinanceClient) SignedGET(ctx context.Context, path string, query url.Values, out any) error {
	if c == nil || c.APIKey == "" || c.APISecret == "" {
		return fmt.Errorf("Binance client 缺少 API credentials")
	}
	params := url.Values{}
	for key, values := range query {
		for _, value := range values {
			params.Add(key, value)
		}
	}
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	encoded := params.Encode()
	h := hmac.New(sha256.New, []byte(c.APISecret))
	_, _ = h.Write([]byte(encoded))
	params.Set("signature", hex.EncodeToString(h.Sum(nil)))

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.BaseURL, "/")+path+"?"+params.Encode(), nil)
	if err != nil {
		return fmt.Errorf("创建 Binance 请求失败: %w", err)
	}
	request.Header.Set("X-MBX-APIKEY", c.APIKey)
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("请求 Binance 失败: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("读取 Binance 响应失败: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Binance HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("解析 Binance 响应失败: %w", err)
	}
	return nil
}

type binanceSpotTradeResponse struct {
	ID              int64  `json:"id"`
	OrderID         int64  `json:"orderId"`
	Symbol          string `json:"symbol"`
	Price           string `json:"price"`
	Quantity        string `json:"qty"`
	QuoteQuantity   string `json:"quoteQty"`
	Commission      string `json:"commission"`
	CommissionAsset string `json:"commissionAsset"`
	Time            int64  `json:"time"`
	IsBuyer         bool   `json:"isBuyer"`
}

func (c *BinanceClient) FetchSpotTrades(ctx context.Context, symbol string, from, to time.Time) ([]BinanceEvent, error) {
	limit := c.TradePageSize
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	params := url.Values{"symbol": []string{strings.ToUpper(symbol)}, "limit": []string{strconv.Itoa(limit)}}
	if !from.IsZero() {
		params.Set("startTime", strconv.FormatInt(from.UnixMilli(), 10))
	}
	if !to.IsZero() {
		params.Set("endTime", strconv.FormatInt(to.UnixMilli(), 10))
	}
	var events []BinanceEvent
	var nextID int64
	for {
		pageParams := cloneValues(params)
		if nextID > 0 {
			pageParams.Set("fromId", strconv.FormatInt(nextID, 10))
		}
		var rows []binanceSpotTradeResponse
		if err := c.SignedGET(ctx, "/api/v3/myTrades", pageParams, &rows); err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			break
		}
		for _, row := range rows {
			event, err := spotTradeEvent(row)
			if err != nil {
				return nil, err
			}
			events = append(events, event)
		}
		lastID := rows[len(rows)-1].ID
		if len(rows) < limit || lastID < 1 || lastID+1 <= nextID {
			break
		}
		nextID = lastID + 1
	}
	return events, nil
}

func spotTradeEvent(row binanceSpotTradeResponse) (BinanceEvent, error) {
	base, quote, ok := SplitBinanceSymbol(row.Symbol)
	if !ok {
		return BinanceEvent{}, fmt.Errorf("无法拆分 Binance API 交易对: %s", row.Symbol)
	}
	quantity, err := decimal.NewFromString(row.Quantity)
	if err != nil {
		return BinanceEvent{}, fmt.Errorf("解析 API quantity 失败: %w", err)
	}
	quoteQuantity, err := decimal.NewFromString(row.QuoteQuantity)
	if err != nil {
		return BinanceEvent{}, fmt.Errorf("解析 API quoteQty 失败: %w", err)
	}
	price, err := decimal.NewFromString(row.Price)
	if err != nil {
		return BinanceEvent{}, fmt.Errorf("解析 API price 失败: %w", err)
	}
	fee, err := decimal.NewFromString(row.Commission)
	if err != nil {
		return BinanceEvent{}, fmt.Errorf("解析 API commission 失败: %w", err)
	}
	return BinanceEvent{
		Source: "binance-api", EventID: strconv.FormatInt(row.ID, 10), OrderID: strconv.FormatInt(row.OrderID, 10), EventType: "spot_trade",
		Symbol: strings.ToUpper(row.Symbol), BaseAsset: base, QuoteAsset: quote,
		Side: map[bool]string{true: "buy", false: "sell"}[row.IsBuyer],
		Time: time.UnixMilli(row.Time), Quantity: quantity, QuoteQuantity: quoteQuantity,
		Price: price, Fee: fee, FeeAsset: strings.ToUpper(row.CommissionAsset),
	}, nil
}

func cloneValues(values url.Values) url.Values {
	clone := url.Values{}
	for key, items := range values {
		clone[key] = append([]string(nil), items...)
	}
	return clone
}

type binanceDepositResponse struct {
	ID         string `json:"id"`
	Amount     string `json:"amount"`
	Coin       string `json:"coin"`
	Status     int    `json:"status"`
	InsertTime int64  `json:"insertTime"`
	TxID       string `json:"txId"`
}

func (c *BinanceClient) FetchDeposits(ctx context.Context, from, to time.Time) ([]BinanceEvent, error) {
	limit := walletPageSize(c, 1000)
	var events []BinanceEvent
	for offset := 0; ; offset += limit {
		params := timeWindowParams(from, to)
		params.Set("offset", strconv.Itoa(offset))
		params.Set("limit", strconv.Itoa(limit))
		var rows []binanceDepositResponse
		if err := c.SignedGET(ctx, "/sapi/v1/capital/deposit/hisrec", params, &rows); err != nil {
			return nil, err
		}
		for _, row := range rows {
			amount, err := decimal.NewFromString(row.Amount)
			if err != nil {
				return nil, fmt.Errorf("解析 deposit amount 失败: %w", err)
			}
			events = append(events, BinanceEvent{Source: "binance-api", EventID: firstNonEmpty(row.TxID, row.ID), EventType: "deposit", BaseAsset: strings.ToUpper(row.Coin), Quantity: amount, Time: time.UnixMilli(row.InsertTime), Raw: map[string]string{"status": strconv.Itoa(row.Status)}})
		}
		if len(rows) < limit {
			break
		}
	}
	return events, nil
}

type binanceWithdrawResponse struct {
	ID             string `json:"id"`
	Amount         string `json:"amount"`
	TransactionFee string `json:"transactionFee"`
	Coin           string `json:"coin"`
	ApplyTime      string `json:"applyTime"`
	TxID           string `json:"txId"`
}

func (c *BinanceClient) FetchWithdrawals(ctx context.Context, from, to time.Time) ([]BinanceEvent, error) {
	limit := walletPageSize(c, 1000)
	var events []BinanceEvent
	for offset := 0; ; offset += limit {
		params := timeWindowParams(from, to)
		params.Set("offset", strconv.Itoa(offset))
		params.Set("limit", strconv.Itoa(limit))
		var rows []binanceWithdrawResponse
		if err := c.SignedGET(ctx, "/sapi/v1/capital/withdraw/history", params, &rows); err != nil {
			return nil, err
		}
		for _, row := range rows {
			amount, err := decimal.NewFromString(row.Amount)
			if err != nil {
				return nil, fmt.Errorf("解析 withdrawal amount 失败: %w", err)
			}
			fee, err := decimal.NewFromString(row.TransactionFee)
			if err != nil {
				return nil, fmt.Errorf("解析 withdrawal fee 失败: %w", err)
			}
			applyTime, parseErr := parseBinanceAPITime(row.ApplyTime)
			if parseErr != nil {
				return nil, fmt.Errorf("解析 withdrawal applyTime 失败: %w", parseErr)
			}
			events = append(events, BinanceEvent{Source: "binance-api", EventID: firstNonEmpty(row.TxID, row.ID), EventType: "withdrawal", BaseAsset: strings.ToUpper(row.Coin), Quantity: amount, Fee: fee, FeeAsset: strings.ToUpper(row.Coin), Time: applyTime, Raw: map[string]string{"applyTime": row.ApplyTime}})
		}
		if len(rows) < limit {
			break
		}
	}
	return events, nil
}

func (c *BinanceClient) FetchFiatOrders(ctx context.Context, from, to time.Time) ([]BinanceEvent, error) {
	type fiatOrder struct {
		OrderNo         string `json:"orderNo"`
		CreateTime      int64  `json:"createTime"`
		FiatCurrency    string `json:"fiatCurrency"`
		Amount          string `json:"amount"`
		CryptoCurrency  string `json:"cryptoCurrency"`
		CryptoAmount    string `json:"cryptoAmount"`
		TransactionType string `json:"transactionType"`
		Status          string `json:"status"`
	}
	rowsPerPage := walletPageSize(c, 500)
	var events []BinanceEvent
	for page := 1; ; page++ {
		params := url.Values{}
		if !from.IsZero() {
			params.Set("beginTime", strconv.FormatInt(from.UnixMilli(), 10))
		}
		if !to.IsZero() {
			params.Set("endTime", strconv.FormatInt(to.UnixMilli(), 10))
		}
		params.Set("page", strconv.Itoa(page))
		params.Set("rows", strconv.Itoa(rowsPerPage))
		var response struct {
			Data []fiatOrder `json:"data"`
		}
		if err := c.SignedGET(ctx, "/sapi/v1/fiat/orders", params, &response); err != nil {
			return nil, err
		}
		for _, row := range response.Data {
			if !strings.EqualFold(row.Status, "SUCCESS") || !strings.EqualFold(row.TransactionType, "BUY") {
				continue
			}
			quantity, err := decimal.NewFromString(row.CryptoAmount)
			if err != nil {
				return nil, fmt.Errorf("解析 C2C cryptoAmount 失败: %w", err)
			}
			amount, err := decimal.NewFromString(row.Amount)
			if err != nil {
				return nil, fmt.Errorf("解析 C2C amount 失败: %w", err)
			}
			price := decimal.Zero
			if !quantity.IsZero() {
				price = amount.Div(quantity)
			}
			events = append(events, BinanceEvent{
				Source: "binance-api", EventID: row.OrderNo, EventType: "fiat_buy",
				BaseAsset: strings.ToUpper(row.CryptoCurrency), QuoteAsset: strings.ToUpper(row.FiatCurrency), Side: "buy",
				Time: time.UnixMilli(row.CreateTime), Quantity: quantity, QuoteQuantity: amount, Price: price,
				Raw: map[string]string{"status": row.Status, "transactionType": row.TransactionType},
			})
		}
		if len(response.Data) < rowsPerPage {
			break
		}
	}
	return events, nil
}

func walletPageSize(client *BinanceClient, max int) int {
	size := client.WalletPageSize
	if size <= 0 || size > max {
		return max
	}
	return size
}

func parseBinanceAPITime(value string) (time.Time, error) {
	if millis, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.UnixMilli(millis), nil
	}
	return time.ParseInLocation("2006-01-02 15:04:05", value, time.Local)
}

func timeWindowParams(from, to time.Time) url.Values {
	params := url.Values{}
	if !from.IsZero() {
		params.Set("startTime", strconv.FormatInt(from.UnixMilli(), 10))
	}
	if !to.IsZero() {
		params.Set("endTime", strconv.FormatInt(to.UnixMilli(), 10))
	}
	return params
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
