package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestSignedGETIncludesAPIKeyTimestampAndSignature(t *testing.T) {
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		if r.Header.Get("X-MBX-APIKEY") != "key" {
			t.Fatal("缺少 API key")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"serverTime":123}`)
	}))
	defer server.Close()

	client := &BinanceClient{BaseURL: server.URL, APIKey: "key", APISecret: "secret", HTTPClient: server.Client()}
	var out map[string]any
	if err := client.SignedGET(context.Background(), "/api/v3/time", nil, &out); err != nil {
		t.Fatal(err)
	}
	if gotQuery.Get("timestamp") == "" || gotQuery.Get("signature") == "" {
		t.Fatal("缺少签名参数")
	}
}

func TestNewBinanceClientFromEnvRequiresCredentials(t *testing.T) {
	t.Setenv("BINANCE_API_KEY", "")
	t.Setenv("BINANCE_API_SECRET", "")
	if _, err := NewBinanceClientFromEnv(); err == nil {
		t.Fatal("缺少凭据时应报错")
	}
}

func TestFetchSpotTradesPaginatesAndPreservesFeeAsset(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/api/v3/myTrades" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		fromID := r.URL.Query().Get("fromId")
		var body []map[string]any
		if fromID == "3" {
			body = []map[string]any{}
		} else if fromID == "" {
			body = []map[string]any{{"id": 1, "orderId": 10, "symbol": "BTCUSDT", "price": "65000", "qty": "0.01", "quoteQty": "650", "commission": "0.65", "commissionAsset": "USDT", "time": 1760000000000, "isBuyer": true}}
		} else {
			body = []map[string]any{{"id": 2, "orderId": 11, "symbol": "BTCUSDT", "price": "66000", "qty": "0.01", "quoteQty": "660", "commission": "0.00001", "commissionAsset": "BTC", "time": 1760000001000, "isBuyer": false}}
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer server.Close()

	client := &BinanceClient{BaseURL: server.URL, APIKey: "key", APISecret: "secret", HTTPClient: server.Client(), TradePageSize: 1}
	events, err := client.FetchSpotTrades(context.Background(), "BTCUSDT", timeFromMillis(1760000000000), timeFromMillis(1760000002000))
	if err != nil {
		t.Fatal(err)
	}
	if requests < 2 || len(events) != 2 {
		t.Fatalf("requests=%d events=%d", requests, len(events))
	}
	if events[0].FeeAsset != "USDT" || events[1].FeeAsset != "BTC" {
		t.Fatalf("fee assets=%q,%q", events[0].FeeAsset, events[1].FeeAsset)
	}
}

func timeFromMillis(value int64) time.Time {
	return time.UnixMilli(value)
}

func TestSignedGETReturnsHTTPErrorWithoutSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"code":-2015,"msg":"invalid api-key"}`)
	}))
	defer server.Close()
	client := &BinanceClient{BaseURL: server.URL, APIKey: "key", APISecret: "secret", HTTPClient: server.Client()}
	var out map[string]any
	if err := client.SignedGET(context.Background(), "/api/v3/time", nil, &out); err == nil {
		t.Fatal("401 应返回错误")
	}
}

func TestFetchDepositsMapsDepositEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sapi/v1/capital/deposit/hisrec" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		_, _ = io.WriteString(w, `[{"id":"deposit-1","amount":"1000","coin":"USDT","status":1,"insertTime":1760000000000,"txId":"tx-deposit-1"}]`)
	}))
	defer server.Close()
	client := &BinanceClient{BaseURL: server.URL, APIKey: "key", APISecret: "secret", HTTPClient: server.Client()}
	events, err := client.FetchDeposits(context.Background(), time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].EventType != "deposit" || events[0].EventID != "tx-deposit-1" || events[0].BaseAsset != "USDT" {
		t.Fatalf("events=%+v", events)
	}
}

func TestFetchWithdrawalsMapsFee(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sapi/v1/capital/withdraw/history" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		_, _ = io.WriteString(w, `[{"id":"withdraw-1","amount":"0.01","transactionFee":"0.0001","coin":"BTC","applyTime":"2026-09-01 12:00:00","txId":"tx-withdraw-1"}]`)
	}))
	defer server.Close()
	client := &BinanceClient{BaseURL: server.URL, APIKey: "key", APISecret: "secret", HTTPClient: server.Client()}
	events, err := client.FetchWithdrawals(context.Background(), time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].EventType != "withdrawal" || events[0].FeeAsset != "BTC" || !events[0].Fee.Equal(decimal.RequireFromString("0.0001")) {
		t.Fatalf("events=%+v", events)
	}
}

func TestFetchFiatOrdersMapsC2CEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sapi/v1/fiat/orders" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"data":[{"orderNo":"c2c-1","createTime":1760000000000,"fiatCurrency":"CNY","amount":"7200","cryptoCurrency":"USDT","cryptoAmount":"1000","transactionType":"BUY","status":"SUCCESS"}]}`)
	}))
	defer server.Close()
	client := &BinanceClient{BaseURL: server.URL, APIKey: "key", APISecret: "secret", HTTPClient: server.Client()}
	events, err := client.FetchFiatOrders(context.Background(), time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].EventType != "fiat_buy" || events[0].EventID != "c2c-1" || events[0].BaseAsset != "USDT" || !events[0].Quantity.Equal(decimal.RequireFromString("1000")) || !events[0].QuoteQuantity.Equal(decimal.RequireFromString("7200")) {
		t.Fatalf("events=%+v", events)
	}
}
