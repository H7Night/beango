package service

import (
	"beango/model"
	"beango/utils"
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

// RunCLI 执行 CLI 转换
// sourceType: "alipay" 或 "wechat"
// filePath: 输入文件路径
// outputDir: 输出目录
// merge: 是否合并模式
func RunCLI(sourceType, filePath, outputDir string, merge bool) error {
	// 1. 加载 account_map
	if err := model.LoadAccountMap(); err != nil {
		return fmt.Errorf("加载账户映射失败: %w", err)
	}

	// 2. 初始化输出目录
	if err := utils.InitOutputDir(); err != nil {
		return fmt.Errorf("初始化输出目录失败: %w", err)
	}

	// 3. 解析文件
	var entries []string
	var count [5]int
	var err error

	switch sourceType {
	case "alipay":
		entries, count, err = parseAlipayFile(filePath)
	case "wechat":
		entries, count, err = parseWechatFile(filePath)
	default:
		return fmt.Errorf("不支持的类型: %s（仅支持 alipay 或 wechat）", sourceType)
	}
	if err != nil {
		return fmt.Errorf("解析文件失败: %w", err)
	}

	// 4. 确定输出目录
	outDir := outputDir
	if outDir == "" {
		outDir = model.GetConfigString("outputFolder", model.DefaultOutputFolder)
	}

	// 5. 转换并输出
	if err := TransToBeancount(entries, outDir, merge); err != nil {
		return fmt.Errorf("转换 beancount 失败: %w", err)
	}

	// 6. 输出统计
	fmt.Printf("\n=== 转换完成 ===\n")
	fmt.Printf("支出: %d  收入: %d  转账: %d  未识别: %d  跳过: %d\n",
		count[0], count[1], count[2], count[3], count[4])
	fmt.Printf("输出目录: %s\n", outDir)

	return nil
}

// parseAlipayFile 解析支付宝 CSV 文件（GBK → UTF8 → CSV rows → TransAlipay）
func parseAlipayFile(filePath string) ([]string, [5]int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, [5]int{}, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	content, err := utils.ConvertGBKtoUTF8withBom(file)
	if err != nil {
		return nil, [5]int{}, fmt.Errorf("GBK 转换失败: %w", err)
	}

	reader := csv.NewReader(bufio.NewReader(bytes.NewReader(content)))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	var records [][]string
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Println("Skip wrong row", err)
			continue
		}
		if len(row) < 5 {
			continue
		}
		records = append(records, row)
	}

	return TransAlipay(records)
}

// parseWechatFile 解析微信 Excel 文件（清洗 → TransWechat）
func parseWechatFile(filePath string) ([]string, [5]int, error) {
	srcExcel, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, [5]int{}, fmt.Errorf("打开 Excel 失败: %w", err)
	}

	sheetName := srcExcel.GetSheetName(0)
	rows, err := srcExcel.GetRows(sheetName)
	if err != nil {
		return nil, [5]int{}, fmt.Errorf("读取工作表失败: %w", err)
	}

	// 清洗数据
	var cleanRows [][]string
	for _, row := range rows {
		var cleanRow []string
		skip := true
		for _, cell := range row {
			val := strings.TrimSpace(cell)
			if val != "" {
				skip = false
			}
			cleanRow = append(cleanRow, val)
		}
		if skip || len(cleanRow) < 10 || (cleanRow[2] == "" && cleanRow[3] == "" && cleanRow[4] == "") {
			continue
		}
		cleanRows = append(cleanRows, cleanRow)
	}

	return TransWechat(cleanRows)
}
