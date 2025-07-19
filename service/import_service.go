package service

import (
	"beango/model"
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// UTF-8 BOM 的字节序列
var utf8Bom = []byte{0xEF, 0xBB, 0xBF}
var count = [5]int{0, 0, 0, 0, 0} //支出、收入、转账、undefined、不记录

const convertAliCSV = "output/convert-alipay.csv"
const convertWecCSV = "output/convert-wechat.csv"

// TODO 通用导入
func ImportCSV(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file:" + err.Error()})
		return
	}
	baseFile, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file" + err.Error()})
		return
	}
	defer baseFile.Close()

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, baseFile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file: " + err.Error()})
		return
	}
	raw := buf.Bytes()

	fileType := ""
	if IsGBK(raw) {
		fileType = "alipay"
	} else if IsUTF8(raw) {
		fileType = "wechat"
	}

	if fileType == "" {
		log.Println("analyse file by row")
		reader := bufio.NewReader(bytes.NewReader(raw))
		var lines []string
		for i := 0; i < 30; i++ {
			line, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				break
			}
			lines = append(lines, line)
			if err == io.EOF {
				break
			}
		}
		alipayIdent := "支付宝（中国）网络技术有限公司"
		wechatIdent := "微信支付账单明细列表"

		if len(lines) >= 24 && strings.Contains(lines[23], alipayIdent) {
			fileType = "alipay"
		} else if len(lines) >= 16 && strings.Contains(lines[15], wechatIdent) {
			fileType = "wechat"
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无法识别上传文件类型（支付宝或微信）"})
			return
		}
	}

	// 还原 Request.Body 内容给子方法使用
	c.Request.Body = io.NopCloser(bytes.NewReader(raw))
	// 调用对应处理方法
	switch fileType {
	case "alipay":
		ImportAlipayCSV(c)
	case "wechat":
		ImportWechatCSV(c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "未知文件类型"})
		return
	}
}

// 导入 支付宝 账单
func ImportAlipayCSV(c *gin.Context) {
	err := model.LoadAccountMapFromDB()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file:" + err.Error()})
		return
	}
	baseFile, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file: " + err.Error()})
		return
	}
	defer baseFile.Close()
	// 转换成utf8并添加bom
	content, _ := ConvertGBKtoUTF8withBom(baseFile)

	// 保存转换后的内容
	targetFile, _ := os.Create(convertAliCSV)
	defer targetFile.Close()
	if _, err := targetFile.Write(content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write file: " + err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert to beancount: " + err.Error()})
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

// 导入 微信 账单
func ImportWechatCSV(c *gin.Context) {
	err := model.LoadAccountMapFromDB()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to get file" + err.Error()})
		return
	}
	baseFile, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to open file" + err.Error()})
		return
	}
	defer baseFile.Close()
	content, _ := io.ReadAll(baseFile)

	cleanContent := preCleanContent(string(content))
	// 保存转换后的内容
	targetFile, _ := os.Create(convertWecCSV)
	defer targetFile.Close()
	if _, err := targetFile.Write([]byte(cleanContent)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write file: " + err.Error()})
		return
	}

	reader := csv.NewReader(bufio.NewReader(strings.NewReader(cleanContent)))
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
		if (row[2] == "" && row[3] == "" && row[4] == "") || len(row) < 9 {
			continue
		}
		records = append(records, row)
	}

	res, count, err := TransWechat(records)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	outputFolder := model.GetConfigString("outputFolder", "./output")
	if err := TransToBeancount(res, outputFolder); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert to beancount: " + err.Error()})
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
		return "", fmt.Errorf("failed to walk directory %s: %v", searchDir, err)
	}
	if len(beanFiles) == 0 {
		return "", fmt.Errorf(" no files found matching pattern: %s", searchDir)
	}

	// 对文件按名称倒序排序（如 04.bean, 03.bean...）
	sort.Slice(beanFiles, func(i, j int) bool {
		return beanFiles[i] > beanFiles[j] // 倒序比较
	})

	var builder strings.Builder
	for _, file := range beanFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %v", file, err)
		}
		builder.Write(data)
		if len(data) > 0 && data[len(data)-1] != '\n' {
			builder.WriteByte('\n')
		}

	}
	return builder.String(), nil
}

// SaveImportTransaction 保存解析数据到数据库
func SaveImportTransaction(transaction []model.BeancountTransaction) error {
	db := model.GetDB()
	for _, tx := range transaction {
		var existing model.BeancountTransaction
		err := db.Where("uuid=?", tx.UUID).First(&existing).Error
		if err != nil {
			continue
		}
		if err := db.Create(&tx).Error; err != nil {
			log.Printf("插入失败: uuid=%s, err=%v\n", tx.UUID, err)
			continue
		}
	}
	return nil
}

func IsGBK(data []byte) bool {
	decoder := simplifiedchinese.GBK.NewDecoder()
	_, err := decoder.Bytes(data)
	return err == nil
}

func IsUTF8(data []byte) bool {
	return utf8.Valid(data)
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
