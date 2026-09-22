# P0 导入修复设计（确定性匹配 / 金额解析 / 微信表头校验）

- 日期：2026-09-21
- 状态：已批准，待实现
- 范围：P0-1、P0-3、P0-4（P0-2 指纹去重暂缓）

## 1. 背景

来源是 rustledger（Beancount 的 Rust 实现）导入子系统与 beango 的对照研究。原 P0 清单 4 项：

1. **P0-1 匹配顺序不确定**：`model/mapToSlice`（`model/account_map_model.go:53`）遍历 Go map，Go 的 map 迭代顺序是随机的；两个 service 的匹配循环 `for _, mapping := range accountMap`（`service/transaction_alipay_service.go:128`、`service/transaction_wechat_service.go:116`）依赖其顺序，在多关键词同时命中时「首个命中」的账户在每次进程运行间会变化。
2. **P0-3 金额解析静默吞错**：`amount, _ := strconv.ParseFloat(record.Amount, 64)`（`service/transaction_alipay_service.go:227`、`service/transaction_wechat_service.go:188`）忽略错误，坏金额静默变成 `0.00` 并写入账本（历史上已因此手工修账，见 `scripts/fix_wechat.py`）。
3. **P0-4 微信无表头校验**：`TransWechat` 的表头校验被注释（`service/transaction_wechat_service.go:16-18`），误传文件会静默产出垃圾条目；支付宝已有校验（首行含「交易时间」）。
4. P0-2 无 uuid 条目不去重——本轮不做。

此外，`count`（`service/import_service.go:19`）与 `passAllFlag`（`service/export_service.go:15`）是包级可变全局变量，被并发 Web 导入请求共享，存在数据竞态。

## 2. 目标与非目标

**目标**

- 匹配结果确定：同一输入在任何次运行得到相同账户。
- 金额无法解析时：不写出错误金额，把该行作为诊断返回，其余行照常导入。
- 微信输入做与支付宝一致的表头识别校验。
- 顺带消除 `count` / `passAllFlag` 全局竞态。

**非目标（本轮明确不做）**

- P0-2：无 uuid 条目的内容指纹去重。
- 反向包含匹配 `strings.Contains(mapping.Keyword, record.Counterparty)` 的误匹配问题，以及 `record.Counterparty == ""` 时反向包含恒真的问题（确定性排序无法修复它，留待后续）。
- `LoadCommodityMap` 在逐行循环内重复读盘（`transaction_alipay_service.go:47`、`transaction_wechat_service.go:68`）的性能问题。
- README（`README.md:69-71`）与 `config/beango.yml:12-14` 关于兜底账户的表述漂移。
- Importer 注册表 / 自动识别多账单源。
- 诊断信息在前端 `beango-web` 的展示（本轮只到 API JSON / CLI）。

## 3. 决策记录

| 决策 | 选择 | 理由 |
| --- | --- | --- |
| 范围 | 先做 P0-1+3+4，暂缓 P0-2 | P0-2 会改变输出格式与去重语义，单独设计 |
| P0-1 优先级规则 | 关键词长度降序（越具体越优先） | 与 rustledger「longer patterns first」一致；不改 account_map 格式 |
| P0-3 处理方式 | 跳过该行 + 返回诊断列表 | 不把错金额写入账本；不因单行中止整份导入 |
| P0-4 校验范围 | 仅给微信加校验 | 支付宝已有校验且行为可接受 |
| 落地方式 | 方案 A：结果结构体 | 顺带消除全局竞态；诊断有正规出处 |

## 4. 设计

### 4.1 新增类型（`service/result.go`）

```go
package service

// Diagnostic 记录导入过程中被跳过的异常行，便于人工核查。
type Diagnostic struct {
	Source string   `json:"source"` // alipay | wechat
	Row    int      `json:"row"`    // 处理序列下标（含表头，1-based）
	Reason string   `json:"reason"`
	Raw    []string `json:"raw"`
}

// TransResult 是一次账单转换的完整结果。
type TransResult struct {
	Entries     []string
	Count       [5]int // 支出、收入、转账、undefined、skip
	Diagnostics []Diagnostic
}
```

**已知局限**：`Row` 是「清洗后」记录序列的下标，不是原始文件行号——支付宝在 `import_service.go` 过滤了 `len(row) < 5` 的短行，微信在 `parseWechatFile` 删除了 15 行前言，故两者都会造成下标与原始文件行号偏移。因此诊断同时携带 `Raw` 原文，行号仅作排查线索。

### 4.2 消除全局状态

- 删除 `service/import_service.go:19` 的 `var count` 与 `service/export_service.go:15` 的 `var passAllFlag`。
- 函数签名变更：
  - `func TransAlipay(records [][]string, passAll bool) (*TransResult, error)`
  - `func TransWechat(records [][]string, passAll bool) (*TransResult, error)`
  - `func formatAlipayTransactionEntry(record model.BeancountTransaction, amount float64, passAll bool) string`
  - `func formatWechatTransactionEntry(record model.BeancountTransaction, amount float64, passAll bool) string`
- `format*Entry` 不再自增全局 `count`；调用方按 `record.TransactionType` 推 bucket：
  `支出→0、收入→1、转账→2、其它→3`。
- `parseAlipayFile` / `parseWechatFile` 增加 `passAll bool` 参数并透传给 `Trans*`。
- `RunCLI` 把 `passAll` 透传；`main.go` 无需改动。

### 4.3 P0-1 确定性匹配

- `model/account_map_model.go` 新增纯函数：

  ```go
  // SortBySpecificity 返回按关键词长度降序（同长按关键词升序）排序的副本。
  func SortBySpecificity(m []AccountMap) []AccountMap
  ```

  长度用 `utf8.RuneCountInString(mapping.Keyword)`（中文按字符数而非字节数）。
- `mapToSlice` 改为按 **keyword 升序**排序（供 `accountMapsCache` 与 `GetAllAccountMap` 的 API 列表，顺带让接口列表也确定化）。
- `GetAccountMap()` 返回 `SortBySpecificity(accountMapsCache)`，供两个 service 匹配使用。
- 保留每个 source 各自现有的分层匹配顺序（支付宝：支付方式 asset → 交易对方 → 组合字段；微信：交易对方 → 组合字段）与双向包含逻辑不变；改动只在「同一层内多命中时谁是首个」。

### 4.4 P0-3 金额解析

- 金额解析从 `format*Entry` 上移到 `TransAlipay` / `TransWechat` 的行循环中：

  ```go
  amount, perr := strconv.ParseFloat(record.Amount, 64)
  if perr != nil {
      res.Diagnostics = append(res.Diagnostics, Diagnostic{
          Source: "alipay", // 或 "wechat"
          Row:    rowIdx,    // 1-based，含表头
          Reason: fmt.Sprintf("金额无法解析: %q", record.Amount),
          Raw:    row,
      })
      continue
  }
  ```

- 微信仍保留 `parseWechatRow` 中对 `¥` 前缀的剥离。
- 解析失败的行不进 `Count`、不进 `Entries`；`len(Diagnostics)` 即错误行数。

### 4.5 P0-4 微信表头校验

`TransWechat` 开头（仅微信）：

```go
if len(records) < 2 {
    return nil, errors.New("导入文件不符合微信格式")
}
if !strings.Contains(records[0][0], "交易时间") {
    return nil, errors.New("导入文件不符合微信格式")
}
```

- 传入 `TransWechat` 的 `records[0]` 是清洗后的表头行（原始 xlsx 第 16 行，表头前 15 行说明文字因 `len(cleanRow) < 10` 被 `parseWechatFile` 过滤）。
- 支付宝的校验保持现状，不加固。

### 4.6 对外输出

- Web（`service/import_service.go` 的 `ImportAlipayCSV` / `ImportWechatCSV`）：
  原有 5 个 key（`expensCount`/`incomeCount`/`transsCount`/`undefiCount`/`skipedCount`）不变，
  新增 `"errorCount": len(res.Diagnostics)` 与 `"diagnostics": res.Diagnostics`。
- CLI（`service/cli_service.go` 的 `RunCLI`）：
  计数行之后，若 `len(diags) > 0`，输出诊断段到 stderr（行号 + 原因，附原文摘要）。
- `TransToBeancount` 仍接收 `res.Entries`。

## 5. 测试计划（TDD）

先写失败测试，再实现。

| 项 | 位置 | 断言 |
| --- | --- | --- |
| P0-1 排序 | `model/account_map_model_test.go` | `SortBySpecificity` 按长度降序、同长按 keyword 升序 |
| P0-1 真配置排序 | `model/account_map_model_test.go` | `GetAccountMap()`（真实 `config/account_map.yml`，测试需 `chdir("..")`）中相邻元素的关键词 rune 长度非递增 |
| P0-3 坏金额 | `service/transaction_alipay_service_test.go` | 坏金额行不进 `Entries`，`Diagnostics` 长度 1 且含原因；好行仍产出 |
| P0-4 无表头 | `service/transaction_wechat_service_test.go` | 无表头/行数不足返回错误，错误文案「导入文件不符合微信格式」 |
| P0-4 正常 | `service/transaction_wechat_service_test.go` | 合法表头可正常转换 |
| 回归 | `service/transaction_alipay_service_test.go` | 现有还款/标记用例迁移到新签名与 `TransResult` 后仍通过 |

验收命令：`go test ./...` 全绿；`go build -o bin/beango.exe .` 成功。

## 6. 验收标准

- 相同输入重复导入，账户匹配结果完全一致（无随机差异）。
- 含非法金额的账单：不产出 `0.00` 条目，API/CLI 明确列出该行原因。
- 误把非微信文件当微信导入：返回明确错误，不产出条目。
- `count` / `passAllFlag` 全局变量消失；Web 并发导入无共享可变状态。
- 现有 19 个测试全部通过。

## 7. 后续（本轮外）

- P0-2：无 uuid 内容指纹去重。
- 匹配正确性：空 counterparty 恒真、反向包含误匹配。
- 性能：`LoadCommodityMap` 移出逐行循环。
- 文档：README 兜底账户与 `beango.yml` 对齐。
- 架构：Importer 接口 + 注册表（identify→extract）、诊断前端展示。
