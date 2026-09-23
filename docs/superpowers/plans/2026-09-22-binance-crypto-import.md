# Binance Crypto Import Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 beango 中实现 Binance 现货/C2C/充值提现的 API + CSV 导入，按 USDT 成本和 FIFO 批次生成可校验的 `2-crypto` Beancount 分录，并对 Futures 数据明确诊断而不误解析。

**Architecture:** Binance API 和 CSV 进入同一个 `BinanceEvent` 归一化层；事件按 trade/order/tx ID 去重，再由 FIFO lot engine 生成买入、卖出、手续费、充值和提现分录。CLI 负责同步/导入和输出，API Secret 只来自环境变量，历史 ZIP 中的 Spot CSV 作为集成 fixture，Futures CSV 第一阶段只识别并报告暂不支持。

**Tech Stack:** Go 1.24.2、标准库 `net/http`/`crypto/hmac`/`archive/zip`/`encoding/csv`、现有 beango CLI、Beancount/rledger 校验。

**Spec:** `docs/superpowers/specs/2026-09-22-binance-crypto-import-design.md`

## Global Constraints

- 第一阶段支持 Binance Spot、充值/提现、C2C/法币买币；Futures 单独作为后续阶段。
- `Assets:Crypto` 保留为现有 C2C 买币 CNY 账户；新资产使用 `Assets:Binance:<币种>`。
- 现货币种批次以 USDT 为主要成本基准；C2C 买入 USDT 以 CNY 成本冲减 `Assets:Crypto`。
- API 凭据只从 `BINANCE_API_KEY` / `BINANCE_API_SECRET` 读取；不得进入配置、日志、原始归档或 Beancount metadata。
- HTTP 客户端遵循 `HTTPS_PROXY`/`HTTP_PROXY`，支持 `BINANCE_BASE_URL` 覆盖默认 API 地址。
- 现货 CSV fixture：`test/Binance-现货订单历史记录-*.zip`；合约 CSV fixture：`test/Binance-合约订单历史记录-*.zip`。
- 输出目录：`<output>/<run-date>/<year>/2-crypto/<month>.bean`，原始数据：`<output>/raw/binance/<run-date>/`。
- 重复事件按 `tradeId`/`orderId`/`txId`/C2C order number 去重；不能唯一匹配的 C2C 或未知成本充值必须进入诊断，不得猜测。
- 每个任务结束后运行相关测试；提交信息遵循 `feat(scope): 中文描述` / `fix(scope): 中文描述`。
- 精确金额/数量使用 `github.com/shopspring/decimal v1.4.0`，禁止用 `float64` 做账务计算。

## Review Focus

- Spot CSV 的数量/总额带币种后缀，且交易对包含 `BTCUSDT`、`BTCUSDC`、`ETHUSDC`、`USDCUSDT`、`USDTUSD`：测试必须确认 base/quote 拆分和金额解析没有把单位当数字。
- Spot CSV 与 API 字段不同且 CSV 没有手续费：CSV 不能伪造手续费，API 的 fee asset 必须分别覆盖 quote/base fee。
- 同一 order 的多个 API fills、同一 CSV 重复导入和跨 API/CSV 重复导入：必须幂等，不重复生成分录。
- 合约 CSV 头部与 Spot 相似但含有“用户ID/执行金额/已执行报价金额”：必须识别为 Futures 并输出明确“不支持”，不能生成现货交易。
- FIFO 卖出遇到多批次、部分卖出、未知成本充值：必须正确消耗批次；无法计算时输出诊断，不生成虚假的资本利得。
- FIFO 批次成本货币与卖出 quote 货币不同时：必须诊断且不消耗批次，不能直接相减。

---

### Task 1: Binance 领域模型与 CSV/ZIP 适配器

**Files:**
- Create: `service/binance_model.go`
- Create: `service/binance_csv.go`
- Create: `service/binance_csv_test.go`
- Test fixture: `test/Binance-现货订单历史记录-*.zip`
- Test fixture: `test/Binance-合约订单历史记录-*.zip`

**Interfaces:**
- Consumes: CSV rows from the two Binance ZIP fixtures.
- Produces:
  - `type BinanceEvent struct { Source, EventID, EventType, Symbol, BaseAsset, QuoteAsset, Side string; Time time.Time; Quantity, QuoteQuantity, Price, Fee decimal.Decimal; FeeAsset string; Raw map[string]string }`
  - `type BinanceDiagnostic struct { Source, Reason string; Row int; Raw []string }`
  - `type BinanceParseResult struct { Events []BinanceEvent; Diagnostics []BinanceDiagnostic }`
  - `func ParseBinanceCSV(r io.Reader, sourceName string) (BinanceParseResult, error)`
  - `func ParseBinanceZip(path string) (BinanceParseResult, error)`
  - `func SplitBinanceSymbol(symbol string) (base, quote string, ok bool)`

- [ ] **Step 1: 写失败测试，直接从 ZIP 读取 fixture**

测试必须动态找到 `test/Binance-*.zip`，不能依赖乱码后的 Windows 文件名。按 ZIP 内 CSV 首行判断 Spot/Futures：

```go
func findZipByHeader(t *testing.T, required ...string) string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("..", "test", "Binance-*.zip"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("找不到 Binance fixture: %v", err)
	}
	for _, path := range paths {
		archive, err := zip.OpenReader(path)
		if err != nil { continue }
		for _, file := range archive.File {
			reader, err := file.Open()
			if err != nil { continue }
			header, err := csv.NewReader(reader).Read()
			_ = reader.Close()
			if err != nil { continue }
			joined := strings.Join(header, ",")
			matched := true
			for _, item := range required {
				if !strings.Contains(joined, item) { matched = false; break }
			}
			if matched { _ = archive.Close(); return path }
		}
		_ = archive.Close()
	}
	t.Fatalf("找不到 header=%v 的 fixture", required)
	return ""
}

func findEvent(t *testing.T, events []BinanceEvent, id string) BinanceEvent {
	t.Helper()
	for _, event := range events { if event.EventID == id { return event } }
	t.Fatalf("找不到 event id=%s", id)
	return BinanceEvent{}
}

func assertEventHas(t *testing.T, events []BinanceEvent, symbol, base, quote string) {
	t.Helper()
	for _, event := range events {
		if event.Symbol == symbol && event.BaseAsset == base && event.QuoteAsset == quote { return }
	}
	t.Fatalf("找不到 %s/%s/%s", symbol, base, quote)
}

func parseKnownSpotFixture(t *testing.T) BinanceParseResult {
	t.Helper()
	path := findZipByHeader(t, "交易对", "订单号")
	result, err := ParseBinanceZip(path)
	if err != nil { t.Fatal(err) }
	return result
}

func hasDiagnosticContaining(items []BinanceDiagnostic, text string) bool {
	for _, item := range items { if strings.Contains(item.Reason, text) { return true } }
	return false
}
```

更稳妥的实现方式是测试 ZIP 内每个 CSV 的 header 分类函数：

```go
func TestParseSpotFixture(t *testing.T) {
	path := findZipByHeader(t, "交易对", "订单号")
	result, err := ParseBinanceZip(path)
	if err != nil { t.Fatal(err) }
	if len(result.Events) == 0 { t.Fatal("Spot fixture 应产生事件") }
	if result.Events[0].EventType != "spot_trade" { t.Fatalf("事件类型=%s", result.Events[0].EventType) }
	assertEventHas(t, result.Events, "BTCUSDT", "BTC", "USDT")
	assertEventHas(t, result.Events, "USDCUSDT", "USDC", "USDT")
	assertEventHas(t, result.Events, "ETHUSDC", "ETH", "USDC")
	assertEventHas(t, result.Events, "USDTUSD", "USDT", "USD")
}

func TestParseSpotUnitsAndQuantities(t *testing.T) {
	// fixture 内有 0.00758BTC、649USDC、650.4130426USDT 等带后缀字段。
	result := parseKnownSpotFixture(t)
	trade := findEvent(t, result.Events, "1673595238")
	if trade.BaseAsset != "USDC" || trade.QuoteAsset != "USDT" { t.Fatal("USDCUSDT 拆分错误") }
	if trade.Quantity.String() != "649" { t.Fatalf("数量=%s", trade.Quantity) }
	if trade.QuoteQuantity.String() != "649.07788" { t.Fatalf("报价金额=%s", trade.QuoteQuantity) }
}

func TestParseFuturesFixtureIsUnsupported(t *testing.T) {
	path := findZipByHeader(t, "用户ID", "执行金额")
	result, err := ParseBinanceZip(path)
	if err != nil { t.Fatal(err) }
	if len(result.Events) != 0 { t.Fatal("第一阶段不应生成 Futures 现货事件") }
	if !hasDiagnosticContaining(result.Diagnostics, "Futures") { t.Fatal("应明确报告 Futures 暂不支持") }
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./service/ -run 'TestParseSpotFixture|TestParseSpotUnitsAndQuantities|TestParseFuturesFixtureIsUnsupported' -v`
Expected: 编译失败，领域模型和解析函数尚不存在。

- [ ] **Step 3: 实现领域模型与 ZIP 读取**

使用标准库 `archive/zip`，读取 ZIP 中唯一 CSV；CSV 用 UTF-8 读取，保留原始行到 `Raw`。header 分类：

```go
func classifyBinanceHeader(header []string) string {
	joined := strings.Join(header, ",")
	switch {
	case strings.Contains(joined, "交易对") && strings.Contains(joined, "已执行"):
		return "spot_order_history"
	case strings.Contains(joined, "用户ID") && strings.Contains(joined, "执行金额"):
		return "futures_order_history"
	default:
		return "unknown"
	}
}
```

对 Futures 返回一个 diagnostic：`Futures CSV 暂不支持，仅识别不生成现货分录`。

- [ ] **Step 4: 实现 Spot CSV 数值和交易对解析**

Spot 表的列映射固定为：

```text
时间, 订单号, 交易对, 类型, 方向, 订单价格, 订单金额,
时间, 已执行, 平均价格, 交易总额, 状态
```

使用最长后缀匹配拆分交易对：`USDT`、`USDC`、`BUSD`、`FDUSD`、`USD`、`BTC`、`ETH` 等；优先支持 fixture 中的 `BTCUSDT`、`BTCUSDC`、`ETHUSDT`、`ETHUSDC`、`DOGEUSDT`、`USDCUSDT`、`USDTUSD`。

解析：

- `已执行` 去掉末尾资产符号，得到 base quantity。
- `交易总额` 去掉末尾 quote 资产符号，得到 quote quantity。
- `平均价格` 为每单位价格。
- `订单号` 作为 CSV event ID/order ID。
- 状态只接受 `FILLED`/`Filled`；其他状态进入诊断。
- 该 CSV 没有 fee 字段，不生成手续费，增加 `fee_missing` 诊断 metadata；API 数据有 fee 时才生成手续费。

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./service/ -run 'TestParseSpotFixture|TestParseSpotUnitsAndQuantities|TestParseFuturesFixtureIsUnsupported' -v`
Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add service/binance_model.go service/binance_csv.go service/binance_csv_test.go
git commit -m "feat(binance): 解析现货订单 ZIP 并识别合约文件"
```

---

### Task 2: Binance API 签名客户端与 API 事件适配器

**Files:**
- Create: `service/binance_api.go`
- Create: `service/binance_api_test.go`

**Interfaces:**
- Consumes: `BINANCE_API_KEY`、`BINANCE_API_SECRET`、`BINANCE_BASE_URL`、`HTTPS_PROXY`、Task 1 的 `BinanceEvent`。
- Produces:
  - `type BinanceClient struct { BaseURL, APIKey, APISecret string; HTTPClient *http.Client }`
  - `func NewBinanceClientFromEnv() (*BinanceClient, error)`
  - `func (c *BinanceClient) SignedGET(ctx context.Context, path string, query url.Values, out any) error`
  - `func (c *BinanceClient) FetchSpotTrades(ctx context.Context, symbol string, from, to time.Time) ([]BinanceEvent, error)`
  - `func (c *BinanceClient) FetchDeposits(ctx context.Context, from, to time.Time) ([]BinanceEvent, error)`
  - `func (c *BinanceClient) FetchWithdrawals(ctx context.Context, from, to time.Time) ([]BinanceEvent, error)`
  - `func (c *BinanceClient) FetchFiatOrders(ctx context.Context, from, to time.Time) ([]BinanceEvent, error)`

- [ ] **Step 1: 写 HMAC 签名失败测试**

使用 `httptest.Server` 检查：

```go
func TestSignedGETIncludesAPIKeyTimestampAndSignature(t *testing.T) {
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		if r.Header.Get("X-MBX-APIKEY") != "key" { t.Fatal("缺少 API key") }
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"serverTime":123}`)
	}))
	defer server.Close()
	client := &BinanceClient{BaseURL: server.URL, APIKey: "key", APISecret: "secret", HTTPClient: server.Client()}
	var out map[string]any
	if err := client.SignedGET(context.Background(), "/api/v3/time", nil, &out); err != nil { t.Fatal(err) }
	if gotQuery.Get("timestamp") == "" || gotQuery.Get("signature") == "" { t.Fatal("缺少签名参数") }
}
```

另测：HTTP 451、401、超时、非法 JSON 都返回可读错误，不返回部分事件。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./service/ -run 'TestSignedGET' -v`
Expected: 编译失败，`BinanceClient` 尚不存在。

- [ ] **Step 3: 实现标准库签名请求**

签名算法：对按 Binance 规则编码后的 query string 做 HMAC-SHA256，hex 编码放入 `signature`；所有 signed request 携带毫秒 `timestamp`，可设置 `recvWindow`。请求绝不把 secret 写入错误文本。

`NewBinanceClientFromEnv` 默认 `https://api.binance.com`，HTTP transport 使用 `http.ProxyFromEnvironment`，缺少任一 key/secret 时返回明确错误。

- [ ] **Step 4: 实现 Spot API 分页**

`/api/v3/myTrades` 每个 symbol 分页，使用 `fromId` 或时间游标；把 API 返回的 `id/orderId/symbol/price/qty/quoteQty/commission/commissionAsset/time/isBuyer` 映射到 `BinanceEvent`。同一 order 的多 fill 暂不在 adapter 内合并，由后续归一化层处理。

- [ ] **Step 5: 实现充值、提现、C2C API 映射**

分别映射 deposit/withdraw/fiat 响应的 ID、时间、asset、amount、status、fee、fiat amount、order number。非成功状态生成 diagnostic 或跳过，不生成有效账本事件。

- [ ] **Step 6: mock API 测试**

用 `httptest.Server` 固定 JSON 测试：

- 两页 Spot trades 能完整合并。
- `commissionAsset=USDT` 与 `commissionAsset=BTC` 都保留。
- deposit/withdraw/fiat order 字段映射正确。
- 时间窗超过 API 限制时被拆分。

Run: `go test ./service/ -run 'TestBinanceAPI|TestSignedGET' -v`
Expected: PASS。

- [ ] **Step 7: 提交**

```bash
git add service/binance_api.go service/binance_api_test.go
git commit -m "feat(binance): 增加只读 API 签名客户端与事件适配"
```

---

### Task 3: 事件归一化、去重与 FIFO lot engine

**Files:**
- Create: `service/binance_lot.go`
- Create: `service/binance_lot_test.go`
- Create: `service/binance_normalize.go`
- Create: `service/binance_normalize_test.go`

**Interfaces:**
- Consumes: `[]BinanceEvent` from CSV/API.
- Produces:
  - `type Lot struct { Quantity decimal.Decimal; Cost decimal.Decimal; CostCurrency string; AcquiredAt time.Time; SourceID string }`
  - `type LotBook struct { Accounts map[string][]Lot }`
  - `type GeneratedPosting struct { Account, Currency string; Units, Cost, Price decimal.Decimal; HasCost, HasPrice bool }`
  - `type GeneratedTransaction struct { Date time.Time; Time, Payee, Narration string; Metadata map[string]string; Postings []GeneratedPosting }`
  - `func NormalizeBinanceEvents(events []BinanceEvent) ([]BinanceEvent, []BinanceDiagnostic)`
  - `func ApplyFIFO(book *LotBook, event BinanceEvent) (GeneratedTransaction, []BinanceDiagnostic, error)`

- [ ] **Step 0: 添加精确 decimal 依赖**

Run: `go get github.com/shopspring/decimal@v1.4.0`
Expected: `go.mod`/`go.sum` 增加 `github.com/shopspring/decimal v1.4.0`，后续金额与数量类型统一使用 `decimal.Decimal`。

- [ ] **Step 1: 写 FIFO 失败测试**

覆盖：

```go
func TestApplyFIFOPartialSell(t *testing.T) {
	book := NewLotBook()
	applyBuy(t, book, "BTC", "0.01", "650", "USDT")
	applyBuy(t, book, "BTC", "0.02", "700", "USDT")
	txn, diagnostics, err := ApplyFIFO(book, sellEvent("BTCUSDT", "0.015", "800", "12"))
	if err != nil || len(diagnostics) != 0 { t.Fatal(err, diagnostics) }
	gain := findPosting(t, txn.Postings, "Income:CapitalGains:Crypto")
	if !gain.Units.Equal(decimal.NewFromInt(-2)) { t.Fatalf("FIFO gain=%s", gain.Units) }
}
```

测试文件内定义固定构造 helper，避免测试依赖未声明函数：

```go
func applyBuy(t *testing.T, book *LotBook, asset, quantity, cost, currency string) {
	t.Helper()
	_, diagnostics, err := ApplyFIFO(book, BinanceEvent{
		EventID: "buy-" + quantity + "-" + cost, EventType: "spot_trade", Side: "buy",
		BaseAsset: asset, QuoteAsset: currency,
		Quantity: decimal.RequireFromString(quantity), QuoteQuantity: decimal.RequireFromString(cost),
		Price: decimal.RequireFromString(cost).Div(decimal.RequireFromString(quantity)), Time: time.Unix(0, 0),
	})
	if err != nil || len(diagnostics) != 0 { t.Fatal(err, diagnostics) }
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
	for _, posting := range postings { if posting.Account == account { return posting } }
	t.Fatalf("找不到 posting=%s", account)
	return GeneratedPosting{}
}
```

测试中必须使用精确 Decimal 预期，不使用 float。另测：卖出数量超过库存、未知充值成本、fee 在 base asset、fee 在 quote asset、重复 event ID。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./service/ -run 'TestApplyFIFO|TestNormalizeBinance' -v`
Expected: 编译失败，lot engine 尚不存在。

- [ ] **Step 3: 实现 Decimal lot book**

Task 开始先运行 `go get github.com/shopspring/decimal@v1.4.0`。使用 `decimal.Decimal`，禁止用 `float64` 做金额、数量、成本和收益计算。每个 `Assets:Binance:<asset>` 维护按取得时间排序的 lot queue。

规则：

- buy/deposit with known cost：append lot。
- sell：从最早 lot 开始拆分，生成被处置批次的 `{cost}` 和 gain/loss posting。
- transfer out：消耗 lot 但不生成 gain。
- unknown external deposit：返回 diagnostic，不创建可用于收益计算的虚构 lot。
- event ID 已处理时返回空 transaction，保证幂等。

- [ ] **Step 4: 实现事件归一化**

- 按 `event_id` 去重。
- 按 `(orderId, time)` 合并同一订单多个 fill。
- 规范 side、status、asset 大小写、时间时区。
- 把 CSV 缺少 fee 的事实保留为 diagnostic metadata，不补零手续费分录。

- [ ] **Step 5: 运行 fixture 与 lot 测试**

Run: `go test ./service/ -run 'TestApplyFIFO|TestNormalizeBinance' -v`
Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add service/binance_lot.go service/binance_lot_test.go service/binance_normalize.go service/binance_normalize_test.go
git commit -m "feat(binance): 增加事件去重与 FIFO 成本批次引擎"
```

---

### Task 4: Beancount emitter 与 crypto 输出目录

**Files:**
- Create: `service/binance_export.go`
- Create: `service/binance_export_test.go`
- Modify: `service/export_service.go`
- Modify: `config/beango.yml`

**Interfaces:**
- Consumes: `GeneratedTransaction`、动态 Binance account/commodity 集合。
- Produces: `2-crypto/<year>/<month>.bean`、`2-crypto/00.bean`、raw archive path。

- [ ] **Step 1: 写生成器失败测试**

验证生成内容包含：

- `bill: "binance"`。
- `source_id:` 与 `order_id:`。
- `Assets:Binance:BTC`、`Assets:Binance:USDT`。
- C2C transaction 冲减 `Assets:Crypto`。
- 动态 account/commodity 声明只生成一次。
- 日期、时间排序与现有 exporter 一致。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./service/ -run 'Test(RenderBinanceTransaction|WriteBinanceTransactions)' -v`
Expected: FAIL/编译失败。

- [ ] **Step 3: 实现 crypto 输出分组**

在 `config/beango.yml` 增加：

```yaml
  cryptoFolder: "2-crypto"
```

扩展 exporter 支持 `2-crypto`，不改变 `0-default`/`1-securities` 行为。crypto 月文件按交易时间生成，重复事件使用 source ID 去重。`00.bean` 输出必要的 `open`/`commodity` 声明，避免动态币种未经声明直接导致 bean-check 失败。

- [ ] **Step 4: 实现 metadata 与诊断输出**

保留 `bill: "binance"`、`source_id`、`order_id`、`chain`；CSV fee 缺失、未知成本、Futures 暂不支持、C2C 未匹配写入诊断而不是有效交易。

- [ ] **Step 5: 运行生成器测试**

Run: `go test ./service/ -run 'Test(RenderBinanceTransaction|WriteBinanceTransactions)' -v`
Expected: PASS。

- [ ] **Step 6: 提交**

```bash
git add service/binance_export.go service/binance_export_test.go service/export_service.go config/beango.yml
git commit -m "feat(binance): 输出 2-crypto Beancount 分录与商品声明"
```

---

### Task 5: CLI 编排、API/CSV 入口与 smoke test

**Files:**
- Modify: `main.go`
- Modify: `service/cli_service.go`
- Modify: `service/import_service.go`
- Create: `service/binance_service.go`
- Create: `service/binance_service_test.go`
- Modify: `README.md`
- Modify: `AGENTS.md`

**Interfaces:**
- Consumes: Tasks 1-4 的 parser、client、lot engine、exporter。
- Produces:
  - `beango -type binance -sync --from YYYY-MM-DD --to YYYY-MM-DD --symbols BTCUSDT,ETHUSDT`
  - `beango -type binance <CSV|ZIP> [--from ...] [--to ...] [--dry-run]`
  - API/CLI 诊断统计。

- [ ] **Step 1: 写 CLI 解析失败测试**

测试 flags：

```go
func TestParseBinanceArgs(t *testing.T) {
	got, err := parseArgs([]string{"-type", "binance", "-sync", "--from", "2026-01-01", "--to", "2026-09-22", "--symbols", "BTCUSDT,ETHUSDT"})
	if err != nil { t.Fatal(err) }
	if got.sourceType != "binance" || !got.binanceSync { t.Fatal(got) }
	if len(got.symbols) != 2 { t.Fatal(got.symbols) }
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./... -run TestParseBinanceArgs -v`
Expected: 编译失败或参数未被识别。

- [ ] **Step 3: 扩展 CLI 参数模型**

引入内部 `cliOptions` 结构体并让 `parseArgs` 返回 `(cliOptions, error)`；`main.go`、现有 alipay/wechat 调用方和测试同步迁移。结构体至少包含：

```go
type cliOptions struct {
	sourceType, outputDir string
	merge, passAll bool
	binanceSync, dryRun bool
	from, to string
	symbols []string
	args []string
}
```

在不破坏 alipay/wechat 现有参数的前提下增加：`-sync`、`--from`、`--to`、`--symbols`、`--dry-run`。Binance API sync 不要求位置文件；CSV/ZIP 模式要求文件路径。

- [ ] **Step 4: 实现 binance service 编排**

流程：

1. 加载 env credentials。
2. API 模式按时间窗和 symbol 拉取事件，CSV 模式读取 ZIP/CSV。
3. 归一化、去重、FIFO。
4. 生成诊断。
5. `--dry-run` 只输出预览；否则调用 crypto exporter。
6. 输出统计：事件数、有效交易数、诊断数、FIFO 未匹配数、Futures 拒绝数。

- [ ] **Step 5: 使用真实 ZIP fixture 做集成测试**

Run: `go test ./...`
Expected: 现有 beango 测试 + Binance parser/lot/export/CLI 测试全部通过。

再执行 CLI smoke test：

```powershell
go run . -type binance "test\Binance-现货订单历史记录-202609220730(UTC+8)_0a3c8abd.zip" --dry-run
go run . -type binance "test\Binance-合约订单历史记录-202609220731(UTC+8)_b1b78d9e.zip" --dry-run
```

第二条必须返回 Futures 暂不支持诊断，不生成现货 bean。

- [ ] **Step 6: 更新文档**

README/AGENTS 必须说明：

- API key 环境变量和代理变量。
- Spot CSV 字段格式与 fee 缺失限制。
- 现有 `Assets:Crypto` 是 C2C CNY 账户，新账户是 `Assets:Binance:<币种>`。
- Futures 当前只识别、不生成分录。
- 首次全量同步建议先 `--dry-run`，再校验并输出。

- [ ] **Step 7: 运行最终验证**

Run: `go test ./...`
Run: `go build -o bin/beango.exe .`
Run: `go run . -type binance <spot-zip> --dry-run`
Expected: 全部通过，Futures fixture 明确诊断。

- [ ] **Step 8: 提交**

```bash
git add main.go main_test.go service/cli_service.go service/import_service.go service/binance_service.go service/binance_service_test.go README.md AGENTS.md
git commit -m "feat(binance): 接入 Binance 现货与 C2C 导入 CLI"
```

## Plan Self-Review

Spec coverage：

- API 认证/代理/端点：Task 2。
- API + CSV 双输入：Task 1、Task 2、Task 5。
- Spot/C2C/充值提现：Task 1-5。
- USDT/CNY 成本与 FIFO：Task 3、Task 4。
- `Assets:Crypto` 与 `Assets:Binance`：Task 4、Task 5。
- `2-crypto` 输出、raw archive、诊断、dry-run：Task 4、Task 5。
- Futures 后置且不误解析：Task 1、Task 5。
- 测试 fixture 与错误场景：Review Focus、每个任务的测试步骤。

未覆盖的 Futures 实现、Earn、Convert、空投、价格服务属于 spec 明确的后续阶段，不进入本计划。
