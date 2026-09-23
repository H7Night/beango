package service

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// RenderBinanceTransaction 将内部交易渲染成 Beancount 文本。
func RenderBinanceTransaction(txn GeneratedTransaction) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s * %s %s\n", txn.Date.Format("2006-01-02"), strconv.Quote(txn.Payee), strconv.Quote(txn.Narration))
	if txn.Time != "" {
		fmt.Fprintf(&b, "    time: %s\n", strconv.Quote(txn.Time))
	}
	keys := make([]string, 0, len(txn.Metadata))
	for key := range txn.Metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if txn.Metadata[key] == "" {
			continue
		}
		fmt.Fprintf(&b, "    %s: %s\n", key, strconv.Quote(txn.Metadata[key]))
	}
	for _, posting := range txn.Postings {
		fmt.Fprintf(&b, "    %-42s %s %s", posting.Account, posting.Units.String(), posting.Currency)
		if posting.HasCost {
			fmt.Fprintf(&b, " {%s %s}", posting.Cost.String(), posting.CostCurrency)
		}
		if posting.HasPrice {
			fmt.Fprintf(&b, " @ %s %s", posting.Price.String(), posting.PriceCurrency)
		}
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	return b.String()
}

// WriteBinanceTransactions 按年份/月输出 crypto 交易，并生成账户/商品声明。
func WriteBinanceTransactions(txns []GeneratedTransaction, baseDir, cryptoFolder string) error {
	if cryptoFolder == "" {
		cryptoFolder = "2-crypto"
	}
	byMonth := make(map[string][]GeneratedTransaction)
	assetsByYear := make(map[string]map[string]bool)
	for _, txn := range txns {
		year := txn.Date.Format("2006")
		month := txn.Date.Format("2006-01")
		byMonth[month] = append(byMonth[month], txn)
		if assetsByYear[year] == nil {
			assetsByYear[year] = make(map[string]bool)
		}
		for _, posting := range txn.Postings {
			if strings.HasPrefix(posting.Account, "Assets:Binance:") {
				assetsByYear[year][posting.Account] = true
			}
			if posting.Currency != "" && !isFiatCurrency(posting.Currency) {
				assetsByYear[year]["commodity:"+posting.Currency] = true
			}
		}
	}
	for month, entries := range byMonth {
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].Date.Equal(entries[j].Date) {
				return entries[i].Time < entries[j].Time
			}
			return entries[i].Date.Before(entries[j].Date)
		})
		year := month[:4]
		dir := filepath.Join(baseDir, year, cryptoFolder)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		beanFile := filepath.Join(dir, month[5:]+".bean")
		existing := []string{}
		if content, err := os.ReadFile(beanFile); err == nil {
			existing = parseBeanFileText(string(content))
		} else if !os.IsNotExist(err) {
			return err
		}
		seen := make(map[string]bool)
		for _, entry := range existing {
			if id := extractBinanceSourceID(entry); id != "" {
				seen[id] = true
			}
		}
		combined := append([]string{}, existing...)
		for _, entry := range entries {
			rendered := RenderBinanceTransaction(entry)
			id := extractBinanceSourceID(rendered)
			if id != "" && seen[id] {
				continue
			}
			if id != "" {
				seen[id] = true
			}
			combined = append(combined, rendered)
		}
		sort.SliceStable(combined, func(i, j int) bool { return binanceEntryDate(combined[i]).Before(binanceEntryDate(combined[j])) })
		var b strings.Builder
		for _, entry := range combined {
			b.WriteString(strings.TrimSpace(entry))
			b.WriteString("\n\n")
		}
		if err := os.WriteFile(beanFile, []byte(b.String()), 0644); err != nil {
			return err
		}
	}
	for year, assets := range assetsByYear {
		dir := filepath.Join(baseDir, year, cryptoFolder)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		keys := make([]string, 0, len(assets))
		for key := range assets {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		declarations := make([]string, 0, len(keys)+len(fixedCryptoDeclarations))
		for _, key := range keys {
			declarations = append(declarations, declarationForKey(key))
		}
		declarations = append(declarations, fixedCryptoDeclarations...)
		if err := writeMergedLines(filepath.Join(dir, "accounts.bean"), declarations); err != nil {
			return err
		}
		months, err := listMonthBeans(dir)
		if err != nil {
			return err
		}
		var b strings.Builder
		b.WriteString("include \"accounts.bean\"\n")
		for _, month := range months {
			fmt.Fprintf(&b, "include %q\n", month)
		}
		if err := os.WriteFile(filepath.Join(dir, "00.bean"), []byte(b.String()), 0644); err != nil {
			return err
		}
	}
	return nil
}

var fixedCryptoDeclarations = []string{
	"1970-01-01 open Income:CapitalGains:Crypto",
	"1970-01-01 open Expenses:CapitalLoss:Crypto",
	"1970-01-01 open Expenses:Crypto:Fees:Trading",
	"1970-01-01 open Expenses:Crypto:Fees:Withdrawal",
}

func writeMergedLines(path string, lines []string) error {
	existing := ""
	if content, err := os.ReadFile(path); err == nil {
		existing = string(content)
	} else if !os.IsNotExist(err) {
		return err
	}
	var b strings.Builder
	b.WriteString(existing)
	if existing != "" && !strings.HasSuffix(existing, "\n") {
		b.WriteString("\n")
	}
	for _, line := range lines {
		if line == "" || strings.Contains(existing, line) {
			continue
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0644)
}

func listMonthBeans(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var months []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if matched, _ := regexp.MatchString(`^\d{2}\.bean$`, entry.Name()); matched {
			months = append(months, entry.Name())
		}
	}
	sort.Strings(months)
	return months, nil
}

var binanceSourceIDRe = regexp.MustCompile(`(?m)^\s+source_id:\s+"([^"]+)"`)

func parseBeanFileText(content string) []string {
	content = strings.TrimSpace(strings.ReplaceAll(content, "\r\n", "\n"))
	if content == "" {
		return nil
	}
	return regexp.MustCompile(`\n\s*\n`).Split(content, -1)
}

func extractBinanceSourceID(entry string) string {
	match := binanceSourceIDRe.FindStringSubmatch(entry)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

func binanceEntryDate(entry string) time.Time {
	line := strings.TrimSpace(strings.SplitN(entry, "\n", 2)[0])
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return time.Time{}
	}
	date, err := time.Parse("2006-01-02", fields[0])
	if err != nil {
		return time.Time{}
	}
	return date
}

func declarationForKey(key string) string {
	if strings.HasPrefix(key, "commodity:") {
		return fmt.Sprintf("1970-01-01 commodity %s", strings.TrimPrefix(key, "commodity:"))
	}
	return fmt.Sprintf("1970-01-01 open %s", key)
}

func isFiatCurrency(currency string) bool {
	switch strings.ToUpper(currency) {
	case "CNY", "USD", "EUR", "JPY", "HKD", "GBP":
		return true
	default:
		return false
	}
}
