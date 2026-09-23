package service

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

var binanceAmountRe = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?|\.[0-9]+)([A-Za-z][A-Za-z0-9]*)$`)

var binanceQuoteAssets = []string{
	"FDUSD", "USDT", "USDC", "BUSD", "USD", "BTC", "ETH", "BNB", "EUR", "TRY",
}

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

// SplitBinanceSymbol 使用最长 quote 后缀拆分 Binance 交易对。
func SplitBinanceSymbol(symbol string) (base, quote string, ok bool) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	quotes := append([]string(nil), binanceQuoteAssets...)
	sort.SliceStable(quotes, func(i, j int) bool { return len(quotes[i]) > len(quotes[j]) })
	for _, candidate := range quotes {
		if strings.HasSuffix(symbol, candidate) && len(symbol) > len(candidate) {
			return strings.TrimSuffix(symbol, candidate), candidate, true
		}
	}
	return "", "", false
}

func parseBinanceAmount(value string) (decimal.Decimal, string, error) {
	value = strings.TrimSpace(value)
	match := binanceAmountRe.FindStringSubmatch(value)
	if len(match) != 3 {
		return decimal.Zero, "", fmt.Errorf("金额格式无效: %q", value)
	}
	number, err := decimal.NewFromString(match[1])
	if err != nil {
		return decimal.Zero, "", fmt.Errorf("数值无效 %q: %w", value, err)
	}
	return number, strings.ToUpper(match[2]), nil
}

func parseBinanceTime(value string) (time.Time, error) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Time{}, err
	}
	return time.ParseInLocation("2006-01-02 15:04:05", strings.TrimSpace(value), location)
}

func rawBinanceRow(header, row []string) map[string]string {
	raw := make(map[string]string, len(row))
	for i, value := range row {
		key := fmt.Sprintf("column_%d", i)
		if i < len(header) && header[i] != "" {
			key = header[i]
		}
		raw[key] = value
	}
	return raw
}

// ParseBinanceZip 读取 ZIP 中的 Binance CSV；ZIP 内应包含一个导出 CSV。
func ParseBinanceZip(path string) (BinanceParseResult, error) {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return BinanceParseResult{}, fmt.Errorf("打开 Binance ZIP 失败: %w", err)
	}
	defer archive.Close()
	result := BinanceParseResult{}
	found := false
	for _, file := range archive.File {
		if file.FileInfo().IsDir() || !strings.HasSuffix(strings.ToLower(file.Name), ".csv") {
			continue
		}
		found = true
		reader, err := file.Open()
		if err != nil {
			return BinanceParseResult{}, fmt.Errorf("读取 ZIP CSV 失败: %w", err)
		}
		part, parseErr := ParseBinanceCSV(reader, file.Name)
		_ = reader.Close()
		if parseErr != nil {
			return result, parseErr
		}
		result.Events = append(result.Events, part.Events...)
		result.Diagnostics = append(result.Diagnostics, part.Diagnostics...)
	}
	if !found {
		return BinanceParseResult{}, fmt.Errorf("Binance ZIP 中没有 CSV 文件")
	}
	return result, nil
}

// ParseBinanceCSV 解析 Binance 现货订单历史；合约导出只返回诊断，不产生现货事件。
func ParseBinanceCSV(reader io.Reader, sourceName string) (BinanceParseResult, error) {
	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1
	header, err := csvReader.Read()
	if err != nil {
		return BinanceParseResult{}, fmt.Errorf("读取 Binance CSV 表头失败: %w", err)
	}

	result := BinanceParseResult{}
	switch classifyBinanceHeader(header) {
	case "futures_order_history":
		row := 1
		for {
			values, readErr := csvReader.Read()
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				return result, fmt.Errorf("读取 Binance Futures CSV 第 %d 行失败: %w", row+1, readErr)
			}
			row++
			if len(values) == 0 {
				continue
			}
			result.Diagnostics = append(result.Diagnostics, BinanceDiagnostic{
				Source: sourceName,
				Reason: "Futures CSV 暂不支持，仅识别不生成现货分录",
				Row:    row,
				Raw:    values,
			})
		}
		return result, nil
	case "spot_order_history":
		return parseBinanceSpotRows(csvReader, header, sourceName, result)
	default:
		return result, fmt.Errorf("无法识别 Binance CSV 表头: %s", strings.Join(header, ","))
	}
}

func parseBinanceSpotRows(reader *csv.Reader, header []string, sourceName string, result BinanceParseResult) (BinanceParseResult, error) {
	rowNumber := 1
	for {
		row, err := reader.Read()
		if err == io.EOF {
			return result, nil
		}
		if err != nil {
			return result, fmt.Errorf("读取 Binance Spot CSV 第 %d 行失败: %w", rowNumber+1, err)
		}
		rowNumber++
		if len(row) == 0 || strings.TrimSpace(strings.Join(row, "")) == "" {
			continue
		}
		if len(row) < 12 {
			result.Diagnostics = append(result.Diagnostics, BinanceDiagnostic{Source: sourceName, Reason: "Spot CSV 列数不足", Row: rowNumber, Raw: row})
			continue
		}
		status := strings.ToUpper(strings.TrimSpace(row[11]))
		if status != "FILLED" {
			result.Diagnostics = append(result.Diagnostics, BinanceDiagnostic{Source: sourceName, Reason: "Spot 订单不是 FILLED，已跳过", Row: rowNumber, Raw: row})
			continue
		}
		base, quote, ok := SplitBinanceSymbol(row[2])
		if !ok {
			result.Diagnostics = append(result.Diagnostics, BinanceDiagnostic{Source: sourceName, Reason: "无法拆分 Binance 交易对", Row: rowNumber, Raw: row})
			continue
		}
		quantity, quantityAsset, err := parseBinanceAmount(row[8])
		if err != nil || quantityAsset != base {
			result.Diagnostics = append(result.Diagnostics, BinanceDiagnostic{Source: sourceName, Reason: fmt.Sprintf("已执行数量无效: %v", err), Row: rowNumber, Raw: row})
			continue
		}
		quoteQuantity, quoteAsset, err := parseBinanceAmount(row[10])
		if err != nil || quoteAsset != quote {
			result.Diagnostics = append(result.Diagnostics, BinanceDiagnostic{Source: sourceName, Reason: fmt.Sprintf("交易总额无效: %v", err), Row: rowNumber, Raw: row})
			continue
		}
		price, err := decimal.NewFromString(strings.TrimSpace(row[9]))
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, BinanceDiagnostic{Source: sourceName, Reason: fmt.Sprintf("平均价格无效: %v", err), Row: rowNumber, Raw: row})
			continue
		}
		tradeTime, err := parseBinanceTime(row[0])
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, BinanceDiagnostic{Source: sourceName, Reason: fmt.Sprintf("交易时间无效: %v", err), Row: rowNumber, Raw: row})
			continue
		}
		result.Events = append(result.Events, BinanceEvent{
			Source: sourceName, EventID: strings.TrimSpace(row[1]), OrderID: strings.TrimSpace(row[1]), EventType: "spot_trade", Symbol: strings.ToUpper(strings.TrimSpace(row[2])),
			BaseAsset: base, QuoteAsset: quote, Side: strings.ToLower(strings.TrimSpace(row[4])), Time: tradeTime,
			Quantity: quantity, QuoteQuantity: quoteQuantity, Price: price, Fee: decimal.Zero, FeeAsset: "",
			Raw: rawBinanceRow(header, row),
		})
	}
}
