package service

import (
	"beango/model"
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	zip "github.com/alexmullins/zip"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// UTF-8 BOM 的字节序列
var utf8Bom = []byte{0xEF, 0xBB, 0xBF}
var count = [5]int{0, 0, 0, 0, 0} //支出、收入、转账、undefined、不记录

const convertAli = "output/convert-alipay.csv"
const convertWec = "output/convert-wechat.xlsx"

// ImportAlipayCSV 导入 支付宝 账单
func ImportAlipayCSV(c *gin.Context) {
	err := model.LoadAccountMapFromDB()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "获取文件失败:" + err.Error()})
		return
	}
	baseFile, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "打开文件失败: " + err.Error()})
		return
	}
	defer baseFile.Close()
	// 转换成utf8并添加bom
	content, _ := ConvertGBKtoUTF8withBom(baseFile)

	// 保存转换后的内容
	targetFile, _ := os.Create(convertAli)
	defer targetFile.Close()
	if _, err := targetFile.Write(content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入文件失败: " + err.Error()})
		return
	}

	reader := csv.NewReader(bufio.NewReader(bytes.NewReader(content)))
	reader.FieldsPerRecord = -1 // 不强制所有行字段数一致
	reader.LazyQuotes = true    // 容忍未正确转义的引号

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

	res, count, err := TransAlipay(records)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 输出.bean文件
	outputFolder := model.GetConfigString("outputFolder", "./output")
	if err := TransToBeancount(res, outputFolder); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "转换beancount失败: " + err.Error()})
		return
	}
	// 读取.bean内容
	// data, err := ReadFile(outputFolder)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file" + err.Error()})
	// 	return
	// }
	// fmt.Println(data)
	c.JSON(http.StatusOK, gin.H{
		"expensCount": count[0],
		"incomeCount": count[1],
		"transsCount": count[2],
		"undefiCount": count[3],
		"skipedCount": count[4],
	})

}

// ImportWechatCSV 导入 微信 账单
func ImportWechatCSV(c *gin.Context) {
	err := model.LoadAccountMapFromDB()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "获取文件失败" + err.Error()})
		return
	}
	srcFile, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "打开文件失败" + err.Error()})
		return
	}
	defer srcFile.Close()

	srcExcel, err := excelize.OpenReader(srcFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	sheetName := srcExcel.GetSheetName(0)
	rows, err := srcExcel.GetRows(sheetName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	newExcel := excelize.NewFile()
	newSheet := newExcel.GetSheetName(0)
	rowIndex := 1

	for _, row := range rows {
		var cleanRow []string
		skip := true
		for _, cell := range row {
			val := strings.TrimSpace(cell)
			if val == "/" {
				val = "/"
			}
			if val != "" {
				skip = false
			}
			cleanRow = append(cleanRow, val)
		}
		if skip || len(cleanRow) < 10 || (cleanRow[2] == "" && cleanRow[3] == "" && cleanRow[4] == "") {
			continue
		}
		for colIdx, val := range cleanRow {
			colName, _ := excelize.CoordinatesToCellName(colIdx+1, rowIndex)
			newExcel.SetCellValue(newSheet, colName, val)
		}
		rowIndex++
	}
	// 保存为中间处理文件
	if err := newExcel.SaveAs(convertWec); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存中间Excel失败: " + err.Error()})
		return
	}
	// 第三步：重新读取中间Excel文件用于业务处理
	finalExcel, err := excelize.OpenFile(convertWec)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取中间Excel失败: " + err.Error()})
		return
	}
	finalSheet := finalExcel.GetSheetName(0)
	finalRows, err := finalExcel.GetRows(finalSheet)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取中间Excel内容失败: " + err.Error()})
		return
	}
	var records [][]string
	for _, row := range finalRows {
		records = append(records, row)
	}
	res, count, err := TransWechat(records)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	outputFolder := model.GetConfigString("outputFolder", "./output")
	if err := TransToBeancount(res, outputFolder); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "转换beancount失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"expensCount": count[0],
		"incomeCount": count[1],
		"transsCount": count[2],
		"undefiCount": count[3],
		"skipedCount": count[4],
	})

}

// ConvertGBKtoUTF8withBom 支付宝账单GBK转UTF8
func ConvertGBKtoUTF8withBom(r io.Reader) ([]byte, error) {
	gbkReader := transform.NewReader(bufio.NewReader(r), simplifiedchinese.GBK.NewDecoder())
	// GBK解码器
	utf8Content, err := io.ReadAll(gbkReader)
	if err != nil {
		return nil, err
	}
	return append(utf8Bom, utf8Content...), nil
}

// ReadFile 读取转换输出的.bean文件内容
func ReadFile(basePath string) (string, error) {
	today := time.Now().Format("2006-01-02")
	searchDir := filepath.Join(basePath, today)

	var beanFiles []string

	err := filepath.Walk(searchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".bean") {
			beanFiles = append(beanFiles, path)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("遍历目录失败 %s: %v", searchDir, err)
	}
	if len(beanFiles) == 0 {
		return "", fmt.Errorf("未找到匹配模式的文件: %s", searchDir)
	}

	// 对文件按名称倒序排序（如 04.bean, 03.bean...）
	sort.Slice(beanFiles, func(i, j int) bool {
		return beanFiles[i] > beanFiles[j] // 倒序比较
	})

	var builder strings.Builder
	for _, file := range beanFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("读取文件失败 %s: %v", file, err)
		}
		builder.Write(data)
		if len(data) > 0 && data[len(data)-1] != '\n' {
			builder.WriteByte('\n')
		}

	}
	return builder.String(), nil
}

// 清理不规范数据
func preCleanContent(content string) string {
	content = strings.ReplaceAll(content, "\t", "")
	//content = strings.ReplaceAll(content, "\"", "")
	lines := strings.Split(content, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "/" {
			continue
		}
		cleaned = append(cleaned, line)
	}
	return strings.Join(cleaned, "\n")
}

// 新增：处理带密码zip上传并解压
func ImportAlipayZip(c *gin.Context) {
	// 1. 获取文件和密码
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "获取文件失败:" + err.Error()})
		return
	}
	password := c.PostForm("password")
	if password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少解压密码"})
		return
	}

	outputFolder := model.GetConfigString("outputFolder", "./output")

	// 确保 output 目录存在
	if _, err := os.Stat(outputFolder); os.IsNotExist(err) {
		if err := os.MkdirAll(outputFolder, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法创建output目录"})
			return
		}
	}

	// 2. 保存zip到output目录下，重命名为规范文件名
	timestamp := time.Now().Format("20060102_150405")
	zipSavePath := filepath.Join(outputFolder, fmt.Sprintf("alipay_upload_%s.zip", timestamp))
	src, _ := file.Open()
	defer src.Close()
	outFile, err := os.Create(zipSavePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法创建目标文件: " + err.Error()})
		return
	}
	n, err := io.Copy(outFile, src)
	if err != nil {
		outFile.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "复制文件失败: " + err.Error()})
		return
	}
	outFile.Sync()
	outFile.Close()
	log.Printf("zip文件保存到: %s, 大小: %d 字节", zipSavePath, n)

	log.Printf("正在打开zip文件: %s", zipSavePath)
	r, err := zip.OpenReader(zipSavePath)
	if err != nil {
		log.Printf("打开zip文件失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法打开zip文件"})
		return
	}
	defer r.Close()

	// 创建临时目录
	tmpDir, err := os.MkdirTemp(outputFolder, "alipay-unzip-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法创建临时目录: " + err.Error()})
		return
	}
	log.Printf("解压目录: %s", tmpDir)

	var csvPath string
	for _, f := range r.File {
		log.Printf("发现文件: %s, 加密: %v", f.Name, f.IsEncrypted())
		if filepath.Ext(f.Name) == ".csv" {
			f.SetPassword(password)
			log.Printf("尝试解压文件: %s, 使用密码: %s", f.Name, password)
			rc, err := f.Open()
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "解压csv失败，密码可能错误"})
				return
			}
			defer rc.Close()
			outPath := filepath.Join(tmpDir, filepath.Base(f.Name))
			outFile, _ := os.Create(outPath)
			if _, err := io.Copy(outFile, rc); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "无法创建csv输出文件: " + err.Error()})
				outFile.Close()
				return
			}
			outFile.Close()
			csvPath = outPath
			break
		}
	}
	if csvPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未找到csv文件"})
		return
	}

	// 4. 读取csv内容并走原有逻辑
	csvFile, err := os.Open(csvPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法打开csv文件"})
		return
	}
	defer csvFile.Close()

	content, _ := ConvertGBKtoUTF8withBom(csvFile)
	reader := csv.NewReader(bufio.NewReader(bytes.NewReader(content)))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	var records [][]string
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			continue
		}
		if len(row) < 5 {
			continue
		}
		records = append(records, row)
	}

	res, count, err := TransAlipay(records)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := TransToBeancount(res, outputFolder); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "转换beancount失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"expensCount": count[0],
		"incomeCount": count[1],
		"transsCount": count[2],
		"undefiCount": count[3],
		"skipedCount": count[4],
	})
}
