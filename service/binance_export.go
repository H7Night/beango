package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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
			if posting.Currency != "" {
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
		var b strings.Builder
		for _, entry := range entries {
			b.WriteString(RenderBinanceTransaction(entry))
		}
		if err := os.WriteFile(filepath.Join(dir, month[5:]+".bean"), []byte(b.String()), 0644); err != nil {
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
		var b strings.Builder
		for _, key := range keys {
			if strings.HasPrefix(key, "commodity:") {
				fmt.Fprintf(&b, "1970-01-01 commodity %s\n\n", strings.TrimPrefix(key, "commodity:"))
				continue
			}
			fmt.Fprintf(&b, "1970-01-01 open %s\n", key)
		}
		if err := os.WriteFile(filepath.Join(dir, "00.bean"), []byte(b.String()), 0644); err != nil {
			return err
		}
	}
	return nil
}
