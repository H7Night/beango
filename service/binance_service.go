package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"beango/model"
	"beango/utils"
)

type BinanceCLIOptions struct {
	Sync      bool
	DryRun    bool
	From      string
	To        string
	Symbols   []string
	Input     string
	OutputDir string
}

func RunBinanceCLI(options BinanceCLIOptions) error {
	from, to, err := parseBinanceDateRange(options.From, options.To)
	if err != nil {
		return err
	}
	var events []BinanceEvent
	var diagnostics []BinanceDiagnostic
	if options.Sync {
		client, err := NewBinanceClientFromEnv()
		if err != nil {
			return err
		}
		if len(options.Symbols) == 0 {
			return fmt.Errorf("Binance API 同步必须提供 --symbols")
		}
		ctx := context.Background()
		for _, symbol := range options.Symbols {
			items, fetchErr := client.FetchSpotTrades(ctx, symbol, from, to)
			if fetchErr != nil {
				return fetchErr
			}
			events = append(events, items...)
		}
		for _, fetch := range []func(context.Context, time.Time, time.Time) ([]BinanceEvent, error){client.FetchDeposits, client.FetchWithdrawals, client.FetchFiatOrders} {
			items, fetchErr := fetch(ctx, from, to)
			if fetchErr != nil {
				return fmt.Errorf("Binance API 同步失败，未生成部分结果: %w", fetchErr)
			}
			events = append(events, items...)
		}
	} else {
		if options.Input == "" {
			return fmt.Errorf("Binance CSV/ZIP 导入缺少文件路径")
		}
		result, parseErr := parseBinanceInput(options.Input)
		if parseErr != nil {
			return parseErr
		}
		events, diagnostics = result.Events, result.Diagnostics
	}

	events, normalizeDiagnostics := NormalizeBinanceEvents(events)
	diagnostics = append(diagnostics, normalizeDiagnostics...)
	book := NewLotBook()
	var transactions []GeneratedTransaction
	for _, event := range events {
		txn, items, applyErr := ApplyFIFO(book, event)
		if applyErr != nil {
			return applyErr
		}
		diagnostics = append(diagnostics, items...)
		if len(txn.Postings) > 0 {
			transactions = append(transactions, txn)
		}
	}

	fmt.Printf("Binance 事件: %d  有效分录: %d  诊断: %d\n", len(events), len(transactions), len(diagnostics))
	for _, diagnostic := range diagnostics {
		fmt.Fprintf(os.Stderr, "[%s] 行 %d: %s\n", diagnostic.Source, diagnostic.Row, diagnostic.Reason)
	}
	for _, txn := range transactions {
		if options.DryRun {
			fmt.Print(RenderBinanceTransaction(txn))
		}
	}
	if options.DryRun {
		return nil
	}
	outputDir := options.OutputDir
	if outputDir == "" {
		outputDir = model.GetConfigString("outputFolder", model.DefaultOutputFolder)
	}
	if options.Input != "" {
		if _, archiveErr := utils.ArchiveRawFile("binance", options.Input, outputDir); archiveErr != nil {
			diagnostics = append(diagnostics, BinanceDiagnostic{Source: options.Input, Reason: "原始 Binance 文件归档失败: " + archiveErr.Error()})
		}
	}
	if err := WriteBinanceTransactions(transactions, outputDir, model.GetConfigString("cryptoFolder", "2-crypto")); err != nil {
		return err
	}
	state := map[string]any{"updated_at": time.Now().Format(time.RFC3339), "from": options.From, "to": options.To, "symbols": options.Symbols, "event_count": len(events), "transaction_count": len(transactions)}
	stateBytes, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outputDir, "binance-sync-state.json"), stateBytes, 0644)
}

func parseBinanceInput(path string) (BinanceParseResult, error) {
	if strings.EqualFold(filepath.Ext(path), ".zip") {
		return ParseBinanceZip(path)
	}
	file, err := os.Open(path)
	if err != nil {
		return BinanceParseResult{}, fmt.Errorf("打开 Binance 文件失败: %w", err)
	}
	defer file.Close()
	return ParseBinanceCSV(file, filepath.Base(path))
}

func parseBinanceDateRange(fromValue, toValue string) (time.Time, time.Time, error) {
	var from, to time.Time
	var err error
	if fromValue != "" {
		from, err = time.ParseInLocation("2006-01-02", fromValue, time.Local)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("无效 --from: %w", err)
		}
	}
	if toValue != "" {
		to, err = time.ParseInLocation("2006-01-02", toValue, time.Local)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("无效 --to: %w", err)
		}
		to = to.Add(24*time.Hour - time.Nanosecond)
	}
	return from, to, nil
}
