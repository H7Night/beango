# P0 导入修复 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让账单导入的账户匹配确定、金额解析失败不再静默写错、微信输入做表头校验，并消除 Web 并发导入的全局竞态。

**Architecture:** 引入 `TransResult`/`Diagnostic` 结果结构体替代包级可变全局 `count`/`passAllFlag`；`TransAlipay`/`TransWechat` 改为返回结构体并接收 `passAll`。匹配顺序在 `model.SortBySpecificity` 中固定为「关键词长度降序」。金额解析上移到行循环，失败则记录诊断并跳过该行。微信在入口处做表头识别校验。

**Tech Stack:** Go 1.24+（module `beango`）、gin、excelize、yaml.v3。

## Global Constraints

- 模块 `beango`，Go 1.24.2；构建 `go build -o bin/beango.exe .`。
- 测试：`go test ./...`（基线 19 passed / 6 packages）。测试运行目录是包目录，需要读配置的测试用 `os.Chdir("..")` 回到仓库根。
- 环境 Windows 10 / pwsh7；shell 用 pwsh，避免 Bash 语法。
- 提交信息风格：`fix(scope): 中文描述` 或 `feat(scope): 中文描述`，scope 如 `alipay`/`wechat`/`account_map`；一个 bug/feat 一个提交，不合并。
- 本轮范围仅 P0-1、P0-3、P0-4（见 `docs/superpowers/specs/2026-09-21-p0-import-fixes-design.md`）；不做 P0-2 指纹去重、不改反向包含匹配、不加固支付宝首行、不动 README。
- `Diagnostic.Row` 是「清洗后」序列下标（含表头，1-based），非原始文件行号。
- 每个任务结束提交一次。

---

### Task 1: P0-1 确定性匹配（关键词长度降序）

**Files:**
- Modify: `model/account_map_model.go`
- Test: `model/account_map_model_test.go`（新建）

**Interfaces:**
- Consumes: 无
- Produces: `func SortBySpecificity(m []AccountMap) []AccountMap`；`GetAccountMap()` 返回按具体度排序的切片；`mapToSlice` 返回按 keyword 升序的切片。

- [ ] **Step 1: 写失败测试**

创建 `model/account_map_model_test.go`：

```go
package model

import (
	"os"
	"testing"
	"unicode/utf8"
)

func TestSortBySpecificity(t *testing.T) {
	in := []AccountMap{
		{Keyword: "余额", Account: "A", Type: "asset"},
		{Keyword: "余额宝", Account: "B", Type: "asset"},
		{Keyword: "招商银行", Account: "C", Type: "asset"},
		{Keyword: "招商银行信用卡(2035)", Account: "D", Type: "asset"},
	}
	got := SortBySpecificity(in)
	for i := 0; i < len(got)-1; i++ {
		li := utf8.RuneCountInString(got[i].Keyword)
		lj := utf8.RuneCountInString(got[i+1].Keyword)
		if li < lj {
			t.Fatalf("长度应降序，位置 %d: %q(len=%d) -> %q(len=%d)", i, got[i].Keyword, li, got[i+1].Keyword, lj)
		}
		if li == lj && got[i].Keyword > got[i+1].Keyword {
			t.Fatalf("同长应关键词升序，位置 %d: %q -> %q", i, got[i].Keyword, got[i+1].Keyword)
		}
	}
	if in[0].Keyword != "余额" {
		t.Fatalf("不应修改入参，实际 %q", in[0].Keyword)
	}
}

func TestGetAccountMapSortedBySpecificity(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(".."); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if err := LoadAccountMap(); err != nil {
		t.Fatalf("LoadAccountMap: %v", err)
	}
	got := GetAccountMap()
	if len(got) < 2 {
		t.Fatalf("映射过少: %d", len(got))
	}
	for i := 0; i < len(got)-1; i++ {
		li := utf8.RuneCountInString(got[i].Keyword)
		lj := utf8.RuneCountInString(got[i+1].Keyword)
		if li < lj {
			t.Fatalf("GetAccountMap 未按具体度排序，位置 %d: %q(len=%d) -> %q(len=%d)", i, got[i].Keyword, li, got[i+1].Keyword, lj)
		}
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./model/ -run 'TestSortBySpecificity|TestGetAccountMapSortedBySpecificity' -v`
Expected: 编译失败，`undefined: SortBySpecificity`。

- [ ] **Step 3: 实现**

在 `model/account_map_model.go` 的 import 块加入 `"sort"` 与 `"unicode/utf8"`。

替换 `mapToSlice`：

```go
func mapToSlice(m map[string]AccountMapEntry) []AccountMap {
	result := make([]AccountMap, 0, len(m))
	for keyword, entry := range m {
		result = append(result, AccountMap{
			Keyword: keyword,
			Account: entry.Account,
			Type:    entry.Type,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Keyword < result[j].Keyword })
	return result
}
```

新增函数（放在 `GetAccountMap` 之前）：

```go
// SortBySpecificity 返回按关键词长度降序（同长按关键词升序）排序的副本，
// 使匹配时更具体的关键词优先，且结果确定。
func SortBySpecificity(m []AccountMap) []AccountMap {
	out := make([]AccountMap, len(m))
	copy(out, m)
	sort.SliceStable(out, func(i, j int) bool {
		li := utf8.RuneCountInString(out[i].Keyword)
		lj := utf8.RuneCountInString(out[j].Keyword)
		if li != lj {
			return li > lj
		}
		return out[i].Keyword < out[j].Keyword
	})
	return out
}
```

替换 `GetAccountMap`：

```go
func GetAccountMap() []AccountMap {
	if !accountMapsLoaded {
		_ = LoadAccountMap()
	}
	return SortBySpecificity(accountMapsCache)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./...`
Expected: PASS（含新增两个测试与原有 19 个）。

- [ ] **Step 5: 提交**

```bash
git add model/account_map_model.go model/account_map_model_test.go
git commit -m "fix(account_map): 匹配按关键词长度降序，消除 map 迭代随机顺序"
```

---

### Task 2: 结果结构体 TransResult/Diagnostic，消除全局状态

纯重构，不改变输出行为；现有测试迁移到新签名后必须全绿。

**Files:**
- Create: `service/result.go`
- Modify: `service/transaction_alipay_service.go`
- Modify: `service/transaction_wechat_service.go`
- Modify: `service/export_service.go`
- Modify: `service/import_service.go`
- Modify: `service/cli_service.go`
- Test: `service/transaction_alipay_service_test.go`（迁移）

**Interfaces:**
- Consumes: `model.GetAccountMap()`
- Produces:
  - `type Diagnostic struct { Source string; Row int; Reason string; Raw []string }`
  - `type TransResult struct { Entries []string; Count [5]int; Diagnostics []Diagnostic }`
  - `func bucketOf(transactionType string) int`
  - `func TransAlipay(records [][]string, passAll bool) (*TransResult, error)`
  - `func TransWechat(records [][]string, passAll bool) (*TransResult, error)`
  - `func formatAlipayTransactionEntry(record model.BeancountTransaction, amount float64, passAll bool) string`
  - `func formatWechatTransactionEntry(record model.BeancountTransaction, amount float64, passAll bool) string`
  - `func parseAlipayFile(filePath string, passAll bool) (*TransResult, error)`
  - `func parseWechatFile(filePath string, passAll bool) (*TransResult, error)`

- [ ] **Step 1: 新建 `service/result.go`**

```go
package service

// Diagnostic 记录导入过程中被跳过的异常行，便于人工核查。
// Row 是「清洗后」记录序列的下标（含表头，1-based），非原始文件行号。
type Diagnostic struct {
	Source string   `json:"source"` // alipay | wechat
	Row    int      `json:"row"`
	Reason string   `json:"reason"`
	Raw    []string `json:"raw"`
}

// TransResult 是一次账单转换的完整结果。
type TransResult struct {
	Entries     []string
	Count       [5]int // 支出、收入、转账、undefined、skip
	Diagnostics []Diagnostic
}

// bucketOf 将交易类型映射到 Count 下标：支出 0、收入 1、转账 2、其它 3。
func bucketOf(transactionType string) int {
	switch transactionType {
	case "支出":
		return 0
	case "收入":
		return 1
	case "转账":
		return 2
	default:
		return 3
	}
}
```

- [ ] **Step 2: 改 `service/transaction_alipay_service.go`**

把函数签名与循环改为（保留原有字段提取、匹配、`outerLoop` 与 skip 逻辑不变）：

```go
func TransAlipay(records [][]string, passAll bool) (*TransResult, error) {
	res := &TransResult{}
	if len(records) < 2 {
		log.Println("导入文件不符合支付宝格式")
		return nil, errors.New("导入文件不符合支付宝格式")
	}
	if !strings.Contains(records[0][0], "交易时间") {
		log.Println("导入文件不符合支付宝格式")
		return nil, errors.New("导入文件不符合支付宝格式")
	}
outerLoop:
	for _, row := range records[1:] {
		// ...（字段提取、commodityMap、交易状态、支付方式、备注，全部保持原样）
		// 原本的 count[4]++ 改为 res.Count[4]++
		// ...
		record := model.BeancountTransaction{ /* ...原样... */ }
		amount, _ := strconv.ParseFloat(record.Amount, 64)
		entry := formatAlipayTransactionEntry(record, amount, passAll)
		res.Entries = append(res.Entries, entry)
		res.Count[bucketOf(record.TransactionType)]++
	}
	return res, nil
}
```

`formatAlipayTransactionEntry` 改为：

```go
func formatAlipayTransactionEntry(record model.BeancountTransaction, amount float64, passAll bool) string {
	accountMap := model.GetAccountMap()
	// ...（默认账户与匹配逻辑完全保持原样）...

	date := strings.Split(record.TransactionTime, " ")[0]
	time := strings.Split(record.TransactionTime, " ")[1]
	commodity := record.Commodity

	flag := "*"
	switch record.TransactionType {
	case "支出":
		if expenseAccount == defaultExpense || assetAccount == defaultAsset {
			flag = "!"
		}
	case "收入":
		if incomeAccount == defaultIncome || assetAccount == defaultAsset {
			flag = "!"
		}
	case "转账":
		if toAccount == defaultAsset || fromAccount == defaultAsset {
			flag = "!"
		}
	default:
		flag = "!"
	}
	if passAll {
		flag = "*"
	}

	var entryBuilder strings.Builder
	entryBuilder.WriteString(fmt.Sprintf("%s %s \"%s\" \"%s\"\n", date, flag, record.Counterparty, commodity))
	entryBuilder.WriteString(fmt.Sprintf("    time: \"%s\"\n", time))
	entryBuilder.WriteString(fmt.Sprintf("    uuid: \"%s\"\n", record.UUID))
	entryBuilder.WriteString(fmt.Sprintf("    status: \"%s\"\n", record.TransactionStatus))
	entryBuilder.WriteString(fmt.Sprintf("    bill: \"%s\"\n", record.Source))

	switch record.TransactionType {
	case "支出":
		entryBuilder.WriteString(fmt.Sprintf("    %s    %.2f CNY\n", expenseAccount, amount))
		entryBuilder.WriteString(fmt.Sprintf("    %s   -%.2f CNY\n", assetAccount, amount))
		utils.LogConvert("success", record)
	case "收入":
		entryBuilder.WriteString(fmt.Sprintf("    %s    %.2f CNY\n", assetAccount, amount))
		entryBuilder.WriteString(fmt.Sprintf("    %s   -%.2f CNY\n", incomeAccount, amount))
		utils.LogConvert("success", record)
	case "转账":
		entryBuilder.WriteString(fmt.Sprintf("    chain: \"%s => %s\"\n", fromAccount, toAccount))
		entryBuilder.WriteString(fmt.Sprintf("    %s    %.2f CNY\n", toAccount, amount))
		entryBuilder.WriteString(fmt.Sprintf("    %s   -%.2f CNY\n", fromAccount, amount))
		utils.LogConvert("success", record)
	default:
		entryBuilder.WriteString(fmt.Sprintf("    Equity:Uncategorized    %.2f CNY\n", amount))
		entryBuilder.WriteString(fmt.Sprintf("    Equity:Uncategorized   -%.2f CNY\n", amount))
		utils.LogConvert("undefined", record)
	}
	return entryBuilder.String()
}
```

要点：删除函数内的 `amount, _ := strconv.ParseFloat(...)` 与所有 `count[n]++`；`flag` 逻辑改用参数 `passAll`。

- [ ] **Step 3: 改 `service/transaction_wechat_service.go`**

同样处理：`TransWechat(records [][]string, passAll bool) (*TransResult, error)`；`res.Count[4]++` 取代 `count[4]++`；行内解析 `amount, _ := strconv.ParseFloat(record.Amount, 64)` 并传入 `formatWechatTransactionEntry(record, amount, passAll)`，随后 `res.Count[bucketOf(record.TransactionType)]++`。`formatWechatTransactionEntry` 删除内部 ParseFloat 与 `count[n]++`，`flag` 用参数 `passAll`。

`TransWechat` 结构：

```go
func TransWechat(records [][]string, passAll bool) (*TransResult, error) {
	res := &TransResult{}
	for _, row := range records[1:] {
		record, skip := parseWechatRow(row)
		if skip {
			res.Count[4]++
			utils.LogConvert("skip", row)
			continue
		}
		amount, _ := strconv.ParseFloat(record.Amount, 64)
		entry := formatWechatTransactionEntry(record, amount, passAll)
		res.Entries = append(res.Entries, entry)
		res.Count[bucketOf(record.TransactionType)]++
	}
	return res, nil
}
```

- [ ] **Step 4: 删除全局变量**

- `service/export_service.go`：删除 `var passAllFlag = false`（第 15 行）。
- `service/import_service.go`：删除第 19 行 `var count = [5]int{...}`，删除两处 `passAllFlag = false`（第 23、104 行）。

- [ ] **Step 5: 改 `service/cli_service.go`**

`RunCLI` 内部改为：

```go
	var res *TransResult
	var err error
	switch sourceType {
	case "alipay":
		res, err = parseAlipayFile(filePath, passAll)
	case "wechat":
		res, err = parseWechatFile(filePath, passAll)
	default:
		return fmt.Errorf("不支持的类型: %s（仅支持 alipay 或 wechat）", sourceType)
	}
	if err != nil {
		return fmt.Errorf("解析文件失败: %w", err)
	}
```

删除 `passAllFlag = passAll` 与 `var count [5]int`；`TransToBeancount(res.Entries, outDir, merge)`；统计行改为 `res.Count[0]...res.Count[4]`。

`parseAlipayFile`/`parseWechatFile` 签名改为 `(filePath string, passAll bool) (*TransResult, error)`，结尾分别 `return TransAlipay(records, passAll)` / `return TransWechat(cleanRows, passAll)`。

- [ ] **Step 6: 改 `service/import_service.go`**

`ImportAlipayCSV` / `ImportWechatCSV` 内：

```go
	res, err := TransAlipay(records, false) // 微信处为 TransWechat(records, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// ...
	if err := TransToBeancount(res.Entries, outputFolder, true); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "转换beancount失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"expensCount": res.Count[0],
		"incomeCount": res.Count[1],
		"transsCount": res.Count[2],
		"undefiCount": res.Count[3],
		"skipedCount": res.Count[4],
	})
```

- [ ] **Step 7: 迁移现有测试**

`service/transaction_alipay_service_test.go` 中三处调用改为新签名与结构体：

```go
	res, err := TransAlipay(records, false)
	// res.Entries 代替 entries
	// res.Count[2] 代替 count[2]
```

（`TestTransAlipayRepayment` 的 `count[2] != 24` → `res.Count[2] != 24`；其余用 `res.Entries`。）

- [ ] **Step 8: 运行测试与构建**

Run: `go test ./...`
Expected: PASS（19 个原测试迁移后仍全绿）。
Run: `go build -o bin/beango.exe .`
Expected: 成功。

- [ ] **Step 9: 提交**

```bash
git add service/result.go service/transaction_alipay_service.go service/transaction_wechat_service.go service/export_service.go service/import_service.go service/cli_service.go service/transaction_alipay_service_test.go
git commit -m "refactor(service): 引入 TransResult 结果结构体，移除 count/passAllFlag 全局态"
```

---

### Task 3: P0-3 金额解析失败 → 诊断并跳过

**Files:**
- Modify: `service/transaction_alipay_service.go`
- Modify: `service/transaction_wechat_service.go`
- Test: `service/transaction_alipay_service_test.go`、`service/transaction_wechat_service_test.go`（新建）

**Interfaces:**
- Consumes: `TransResult`、`Diagnostic`、`bucketOf`（Task 2）
- Produces: 金额解析失败行写入 `res.Diagnostics`，不进 `Entries`/`Count`。

- [ ] **Step 1: 写失败测试**

在 `service/transaction_alipay_service_test.go` 追加（`chdirRepoRoot` 若无则先加）：

```go
func chdirRepoRoot(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(".."); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
}

func TestTransAlipayInvalidAmountSkipped(t *testing.T) {
	chdirRepoRoot(t)

	good := alipayRepaymentRow()
	bad := alipayRepaymentRow()
	bad[6] = "abc" // 金额列
	bad[9] = "20260805BADAMOUNT0001"

	records := [][]string{alipayHeaderRow(), good, bad}
	res, err := TransAlipay(records, false)
	if err != nil {
		t.Fatalf("TransAlipay 失败: %v", err)
	}
	if len(res.Entries) != 1 {
		t.Fatalf("期望 1 条有效条目，实际 %d", len(res.Entries))
	}
	if len(res.Diagnostics) != 1 {
		t.Fatalf("期望 1 条诊断，实际 %d", len(res.Diagnostics))
	}
	if res.Diagnostics[0].Row != 3 {
		t.Errorf("期望 Row=3，实际 %d", res.Diagnostics[0].Row)
	}
	if !strings.Contains(res.Diagnostics[0].Reason, "abc") {
		t.Errorf("诊断原因应含原值，实际 %q", res.Diagnostics[0].Reason)
	}
	if res.Diagnostics[0].Source != "alipay" {
		t.Errorf("Source 应为 alipay，实际 %q", res.Diagnostics[0].Source)
	}
}
```

新建 `service/transaction_wechat_service_test.go`：

```go
package service

import (
	"strings"
	"testing"
)

func wechatHeaderRow() []string {
	return []string{"交易时间", "交易类型", "交易对方", "商品", "收/支", "金额(元)", "支付方式", "当前状态", "交易单号", "商户单号", "备注"}
}

func wechatRow(uuid, amount string) []string {
	return []string{
		"2026-09-10 19:13:06", "商户消费", "购票支付", "购票支付", "支出",
		amount, "招商银行信用卡(2035)", "支付成功", uuid, "6133202691000425524", "/",
	}
}

func TestTransWechatInvalidAmountSkipped(t *testing.T) {
	chdirRepoRoot(t)

	records := [][]string{
		wechatHeaderRow(),
		wechatRow("uuid-good", "17"),
		wechatRow("uuid-bad", "abc"),
	}
	res, err := TransWechat(records, false)
	if err != nil {
		t.Fatalf("TransWechat 失败: %v", err)
	}
	if len(res.Entries) != 1 {
		t.Fatalf("期望 1 条有效条目，实际 %d", len(res.Entries))
	}
	if len(res.Diagnostics) != 1 {
		t.Fatalf("期望 1 条诊断，实际 %d", len(res.Diagnostics))
	}
	if res.Diagnostics[0].Source != "wechat" {
		t.Errorf("Source 应为 wechat，实际 %q", res.Diagnostics[0].Source)
	}
	if !strings.Contains(res.Diagnostics[0].Reason, "abc") {
		t.Errorf("诊断原因应含原值，实际 %q", res.Diagnostics[0].Reason)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./service/ -run 'TestTransAlipayInvalidAmountSkipped|TestTransWechatInvalidAmountSkipped' -v`
Expected: FAIL（`len(res.Diagnostics)` 为 0；坏金额行被当作 0.00 产出）。

- [ ] **Step 3: 实现**

`service/transaction_alipay_service.go` 行循环改为带下标并处理错误：

```go
outerLoop:
	for i, row := range records[1:] {
		// ...
		record := model.BeancountTransaction{ /* ...原样... */ }
		amount, perr := strconv.ParseFloat(record.Amount, 64)
		if perr != nil {
			res.Diagnostics = append(res.Diagnostics, Diagnostic{
				Source: "alipay",
				Row:    i + 2,
				Reason: fmt.Sprintf("金额无法解析: %q", record.Amount),
				Raw:    row,
			})
			continue
		}
		entry := formatAlipayTransactionEntry(record, amount, passAll)
		res.Entries = append(res.Entries, entry)
		res.Count[bucketOf(record.TransactionType)]++
	}
```

`service/transaction_wechat_service.go` 同理：

```go
	for i, row := range records[1:] {
		record, skip := parseWechatRow(row)
		if skip {
			res.Count[4]++
			utils.LogConvert("skip", row)
			continue
		}
		amount, perr := strconv.ParseFloat(record.Amount, 64)
		if perr != nil {
			res.Diagnostics = append(res.Diagnostics, Diagnostic{
				Source: "wechat",
				Row:    i + 2,
				Reason: fmt.Sprintf("金额无法解析: %q", record.Amount),
				Raw:    row,
			})
			continue
		}
		entry := formatWechatTransactionEntry(record, amount, passAll)
		res.Entries = append(res.Entries, entry)
		res.Count[bucketOf(record.TransactionType)]++
	}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./...`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add service/transaction_alipay_service.go service/transaction_wechat_service.go service/transaction_alipay_service_test.go service/transaction_wechat_service_test.go
git commit -m "fix(service): 金额无法解析时跳过该行并返回诊断，不再静默写 0.00"
```

---

### Task 4: P0-4 微信表头校验

**Files:**
- Modify: `service/transaction_wechat_service.go`
- Test: `service/transaction_wechat_service_test.go`

**Interfaces:**
- Consumes: `TransWechat`（Task 2/3）
- Produces: 表头不符或行数不足时返回 `errors.New("导入文件不符合微信格式")`。

- [ ] **Step 1: 写失败测试**

在 `service/transaction_wechat_service_test.go` 追加：

```go
func TestTransWechatRejectsMissingHeader(t *testing.T) {
	chdirRepoRoot(t)
	records := [][]string{{"foo", "bar"}, {"a", "b"}}
	_, err := TransWechat(records, false)
	if err == nil || !strings.Contains(err.Error(), "导入文件不符合微信格式") {
		t.Fatalf("期望微信格式错误，实际 %v", err)
	}
}

func TestTransWechatRejectsShortInput(t *testing.T) {
	chdirRepoRoot(t)
	if _, err := TransWechat([][]string{wechatHeaderRow()}, false); err == nil {
		t.Fatal("仅表头应报错")
	}
}

func TestTransWechatAcceptsHeader(t *testing.T) {
	chdirRepoRoot(t)
	res, err := TransWechat([][]string{wechatHeaderRow(), wechatRow("u1", "17")}, false)
	if err != nil {
		t.Fatalf("TransWechat 失败: %v", err)
	}
	if len(res.Entries) != 1 {
		t.Fatalf("期望 1 条，实际 %d", len(res.Entries))
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./service/ -run 'TestTransWechatRejects|TestTransWechatAccepts' -v`
Expected: FAIL（缺少表头校验，`TestTransWechatRejectsMissingHeader` 不报错）。

- [ ] **Step 3: 实现**

在 `service/transaction_wechat_service.go` 的 `TransWechat` 开头加入（并在 import 块加入 `"errors"`）：

```go
	if len(records) < 2 {
		log.Println("导入文件不符合微信格式")
		return nil, errors.New("导入文件不符合微信格式")
	}
	if len(records[0]) == 0 || !strings.Contains(records[0][0], "交易时间") {
		log.Println("导入文件不符合微信格式")
		return nil, errors.New("导入文件不符合微信格式")
	}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./...`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add service/transaction_wechat_service.go service/transaction_wechat_service_test.go
git commit -m "fix(wechat): 增加表头校验，误传文件不再静默产出垃圾条目"
```

---

### Task 5: 对外输出诊断（API JSON + CLI）

**Files:**
- Modify: `service/import_service.go`
- Modify: `service/cli_service.go`
- Modify: `service/result.go`
- Test: `service/result_test.go`（新建）

**Interfaces:**
- Consumes: `TransResult.Diagnostics`（Task 3）
- Produces: `func formatDiagnostics(diags []Diagnostic) string`；Web JSON 新增 `errorCount`/`diagnostics`；CLI 打印诊断段。

- [ ] **Step 1: 写失败测试**

新建 `service/result_test.go`：

```go
package service

import (
	"strings"
	"testing"
)

func TestFormatDiagnostics(t *testing.T) {
	if got := formatDiagnostics(nil); got != "" {
		t.Fatalf("空诊断应返回空串，实际 %q", got)
	}
	diags := []Diagnostic{
		{Source: "alipay", Row: 3, Reason: `金额无法解析: "abc"`},
		{Source: "wechat", Row: 5, Reason: `金额无法解析: "x"`},
	}
	got := formatDiagnostics(diags)
	for _, want := range []string{"alipay", "行 3", `金额无法解析: "abc"`, "wechat", "行 5"} {
		if !strings.Contains(got, want) {
			t.Errorf("输出缺少 %q:\n%s", want, got)
		}
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./service/ -run TestFormatDiagnostics -v`
Expected: 编译失败，`undefined: formatDiagnostics`。

- [ ] **Step 3: 实现**

在 `service/result.go` 追加：

```go
import (
	"fmt"
	"strings"
)

// formatDiagnostics 把诊断渲染为可读文本，无可诊断时返回空串。
func formatDiagnostics(diags []Diagnostic) string {
	if len(diags) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "\n=== 异常行 (%d) ===\n", len(diags))
	for _, d := range diags {
		fmt.Fprintf(&b, "[%s] 行 %d: %s\n", d.Source, d.Row, d.Reason)
	}
	return b.String()
}
```

（把 `import` 放到 `service/result.go` 顶部已有 import 区，与 package 声明配合。）

在 `service/cli_service.go` 的统计输出后追加：

```go
	if s := formatDiagnostics(res.Diagnostics); s != "" {
		fmt.Fprint(os.Stderr, s)
	}
```

在 `service/import_service.go` 两处成功响应 `gin.H` 中追加：

```go
		"errorCount":  len(res.Diagnostics),
		"diagnostics": res.Diagnostics,
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./...`
Expected: PASS。
Run: `go build -o bin/beango.exe .`
Expected: 成功。

- [ ] **Step 5: 提交**

```bash
git add service/result.go service/result_test.go service/import_service.go service/cli_service.go
git commit -m "feat(service): 导入结果对外暴露异常行诊断（API JSON + CLI）"
```

---

## 验收（全部任务完成后）

- [ ] `go test ./...` 全绿。
- [ ] `go build -o bin/beango.exe .` 成功。
- [ ] 相同账单重复导入，账户匹配一致（无随机差异）。
- [ ] 含非法金额的账单：无 `0.00` 条目，API/CLI 列出该行原因。
- [ ] 误传非微信文件：返回「导入文件不符合微信格式」，无条目产出。
- [ ] `grep -rn "passAllFlag\|var count " service/` 无残留。
