# Binance 加密货币导入设计

- 日期：2026-09-22
- 状态：设计已通过，待 spec 审阅与实现计划
- 目标仓库：`beango` + 账本仓库 `../beancount`

## 1. 背景

用户希望把 Binance 的加密货币交易记录完整纳入 Beancount，参考：

- Beancount 加密货币模板：<https://beancount.io/zh/docs/Tips/cryptocurrency-templates-examples>
- Binance Spot API：<https://github.com/binance/binance-spot-api-docs>
- Binance Spot 成交历史导出说明：<https://www.binance.com/en-AU/support/faq/detail/e4ff64f2533f4d23a0b3f8f17f510eab>

当前账本情况：

- `operating_currency` 是 CNY。
- 现有 `Assets:Crypto` 用于 C2C 买币的 CNY 付款。
- 账本尚未建立 BTC/ETH/USDT 等商品批次体系。
- `Income:Invest:Crypto` 已存在，但以 CNY 为限制货币，不直接用于 USDT 资本利得。

## 2. 已确认决策

### 2.1 记账粒度

使用完整商品/批次记账：

- 币种作为 Beancount commodity。
- 买入建立成本批次 `{cost}`。
- 卖出与币币交易按 FIFO 消耗批次。
- 如果 FIFO 批次成本货币与本次卖出 quote 货币不同，不能直接相减；输出诊断并保留批次，等待明确换汇价格或人工处理。
- 记录资本利得、资本损失与交易/提现手续费。

### 2.2 第一阶段范围

第一阶段支持：

- Binance Spot 现货买卖。
- Binance 加密货币充值/提现。
- Binance C2C/法币买币。
- Binance API 同步与 Binance CSV 历史补全。

第二阶段单独支持 Futures：保证金、已实现盈亏、未实现盈亏、资金费率和合约手续费不与现货 FIFO 引擎混合。

### 2.3 成本基准

- 现货交易通常以 USDT 为 quote asset，币种批次以 USDT 计成本。
- C2C 买入 USDT 时，USDT 以 CNY 成本取得，并从既有 `Assets:Crypto` 冲减 CNY。
- USDT 是一个独立 commodity；后续可用 `price USDT CNY` 做 CNY 估值。
- 资本利得第一阶段以交易 quote asset（通常 USDT）表示。

### 2.4 账户结构

保留现有 `Assets:Crypto` 作为 C2C 买币的 CNY 账户，不把新的 Binance 子账户挂在其下，避免历史 posting 与 `leafonly` 约束冲突：

```text
Assets:Crypto                         ; 既有 C2C CNY 付款/结算
Assets:Binance:USDT
Assets:Binance:BTC
Assets:Binance:ETH
Assets:Binance:<其他币种>

Expenses:Crypto:Fees:Trading
Expenses:Crypto:Fees:Withdrawal
Expenses:CapitalLoss:Crypto
Income:CapitalGains:Crypto
```

新资本利得/手续费账户不限制为 CNY，避免与现有 `Income:Invest:Crypto CNY` 冲突。

## 3. 总体架构

```text
Binance API / Binance CSV
        |
        v
Binance 原始事件归一化
        |
        v
tradeId/orderId/txId 去重与 order fill 合并
        |
        v
FIFO 成本批次与资本利得引擎
        |
        v
Beancount 分录生成
        |
        +--> output/<run-date>/<year>/2-crypto/MM.bean
        +--> output/raw/binance/<run-date>/*.json|*.csv
        +--> 诊断报告与同步状态
```

第一阶段只提供 CLI，不把 API Secret 放入 Web UI。

## 4. Binance 数据获取

### 4.1 API 认证与网络

环境变量：

```powershell
$env:BINANCE_API_KEY = "..."
$env:BINANCE_API_SECRET = "..."
$env:HTTPS_PROXY = "http://127.0.0.1:7890" # 需要代理时设置
```

要求：

- 使用 HMAC-SHA256 签名和 `X-MBX-APIKEY`。
- API key 只开启读取权限，不开启交易或提现。
- Secret 不写入配置、日志、原始归档或 Beancount metadata。
- HTTP 客户端遵循 `HTTPS_PROXY`/`HTTP_PROXY` 环境变量。
- 支持 `BINANCE_BASE_URL` 覆盖默认 `https://api.binance.com`，便于地区或测试环境配置。

### 4.2 API 端点

```text
Spot trades:   GET /api/v3/myTrades
Deposits:      GET /sapi/v1/capital/deposit/hisrec
Withdrawals:   GET /sapi/v1/capital/withdraw/history
Fiat/C2C:      GET /sapi/v1/fiat/orders
               GET /sapi/v1/fiat/payments
```

注意：

- `myTrades` 必须指定 `symbol`，不存在一次获取所有历史交易对的单一接口。
- 通过配置的 symbol 列表逐个查询，并用 `fromId`/时间分页。
- 已下架交易对可能无法从当前交易对列表发现，历史全量由 Binance CSV 补全。
- 充值/提现接口按时间窗拆分，避免超过 Binance 的窗口限制和速率限制。
- API 错误、签名错误、时间偏移、权限不足、HTTP 451 都必须明确返回，不写半份结果。

### 4.3 CSV 适配器

支持以下历史补全输入：

- Spot Trade History CSV。
- Deposit History CSV。
- Withdrawal History CSV。
- Fiat/C2C 订单或支付 CSV。

API 和 CSV 都归一化为同一个内部事件模型，不允许两套独立的会计逻辑。

## 5. 统一事件模型

```text
BinanceEvent
  source          api | csv
  event_id        tradeId | orderId | txId | fiatOrderId
  event_type      spot_trade | deposit | withdrawal | fiat_buy
  time
  symbol
  base_asset
  quote_asset
  side            buy | sell
  quantity
  quote_quantity
  price
  fee
  fee_asset
  raw_metadata
```

规则：

- `tradeId`/`txId` 是明细去重键。
- 同一 `orderId` 的多个 fill 合并成一笔交易，但保留所有 tradeId 到 metadata/审计数据。
- C2C 优先使用订单号匹配；缺少共同 ID 时，按时间窗口、CNY 金额、USDT 数量匹配。
- 无法唯一匹配 C2C 付款时输出诊断，不自动猜测对应的 `Assets:Crypto` 记录。

## 6. FIFO 与分录规则

### 6.1 C2C 买入 USDT

```beancount
2026-01-01 * "Binance C2C buy USDT"
    Assets:Binance:USDT       1000 USDT {7200.00 CNY}
    Assets:Crypto             -7200.00 CNY
```

### 6.2 现货买入

```beancount
2026-01-02 * "Binance spot buy BTCUSDT"
    Assets:Binance:BTC          0.01 BTC {650.00 USDT}
    Assets:Binance:USDT       -650.00 USDT
    Expenses:Crypto:Fees:Trading  0.65 USDT
    Assets:Binance:USDT          -0.65 USDT
```

### 6.3 现货卖出

按 FIFO 消耗最早批次：

```beancount
2026-03-01 * "Binance spot sell BTCUSDT"
    Assets:Binance:BTC          -0.01 BTC {650.00 USDT} @ 700.00 USDT
    Assets:Binance:USDT          700.00 USDT
    Income:CapitalGains:Crypto  -50.00 USDT
```

手续费按 Binance 返回的 `commissionAsset` 原币种记录。手续费币种为 BTC 时，减少 BTC 并记录 `Expenses:Crypto:Fees:Trading`；手续费币种为 USDT 时，减少 USDT。

### 6.4 充值/提现

- Binance 内部充值/提现必须区分外部来源和内部钱包转移。
- Binance 到外部钱包的提现不是出售，不产生资本利得；成本随批次转移。
- 提现手续费记录为 `Expenses:Crypto:Fees:Withdrawal`。
- 外部充值没有可靠成本基础时，不自动生成虚构成本；列为待审核事件，要求手工 opening lot 或配置成本。

## 7. CLI 与输出

```powershell
# API 同步
beango -type binance -sync `
  --from 2020-01-01 `
  --to 2026-12-31 `
  --symbols BTCUSDT,ETHUSDT

# CSV 历史补全
beango -type binance "binance-spot-trades.csv" `
  --from 2020-01-01 `
  --to 2026-12-31
```

输出：

```text
<output>/<run-date>/<year>/2-crypto/<month>.bean
<output>/<run-date>/<year>/2-crypto/00.bean
<output>/raw/binance/<run-date>/*.json|*.csv
<output>/binance-sync-state.json
```

要求：

- 支持 `--dry-run`，只生成预览和诊断，不写 bean。
- 重复同步按事件 ID 自动去重。
- 分录带 `bill: "binance"`、`source_id:`、`order_id:`；涉及 C2C/内部转移时带 `chain:`。
- 非法或不完整事件不写入有效交易，统一进入诊断清单并标记 `!`。
- 生成后可执行 `rledger check` 或 Python `bean-check` 验证。

## 8. 非目标与后续阶段

第一阶段不做：

- Futures 保证金、资金费率、已实现/未实现盈亏。
- Earn/质押、闪兑、空投和返佣。
- Web UI 中配置 API Secret。
- 自动在线价格服务；交易本身保留成交价，CNY 估值价格可后续增加。

## 9. 测试要求

- HMAC 签名、时间戳、代理和 API 错误测试。
- API 分页、90 天窗口拆分、重试和速率限制测试。
- CSV/API 归一化结果一致性测试。
- 多 fill 合并、tradeId 去重、重复同步幂等测试。
- C2C 订单号匹配、金额/时间匹配和无法匹配诊断测试。
- FIFO 多批次、部分卖出、手续费为 base/quote asset 的测试。
- 外部充值无成本基础时必须进入诊断，不得生成错误资本利得。
- 使用固定 fixture 生成 `.bean`，再运行 Beancount/rledger 校验。

## 10. 已知风险

- API 不能直接列出所有历史交易对；全量回补必须允许 CSV 补充。
- Binance 地区限制和网络代理可能导致 API 不可用；API 与 CSV 必须都能独立重放。
- C2C 付款与 Binance 订单不一定有相同 ID，自动匹配必须允许人工复核。
- FIFO 只能对有明确成本基础的批次计算收益；未知外部充值不能自动推断。
