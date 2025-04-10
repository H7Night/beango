package service

import (
	"beango/core"
	"beango/model"
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// UTF-8 BOM 的字节序列
var utf8Bom = []byte{0xEF, 0xBB, 0xBF}

func ImportAlipayCSV(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file:" + err.Error()})
		return
	}
	fil, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file: " + err.Error()})
		return
	}
	defer fil.Close()
	// 转换成utf8并添加bom
	content, _ := ConvertGBKtoUTF8withBom(fil)

	// 保存转换后文件
	filename := "convert-alipay.csv"
	targetFile, _ := os.Create(filename)
	defer targetFile.Close()
	targetFile.Write(content)

	reader := csv.NewReader(bufio.NewReader(bytes.NewReader(content)))
	// 不强制所有行字段数一致
	reader.FieldsPerRecord = -1
	// 容忍未正确转义的引号
	reader.LazyQuotes = true

	records := [][]string{}
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			// 跳过脏数据行
			fmt.Println("skip wrong row", err)
			continue
		}
		// 只打印符合要求列数的
		if len(row) < 5 {
			continue
		}
		records = append(records, row)
	}

	TransAlipay(records)
	SaveImportTransaction()
	c.JSON(http.StatusOK, gin.H{"message": "CSV read success"})
}

func ImportWechatCSV(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to get file" + err.Error()})
		return
	}
	fil, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to open file" + err.Error()})
		return
	}
	defer fil.Close()
	content, _ := io.ReadAll(fil)

	// 保存转换后文件
	// filename := "convert-wechat.csv"
	// targetFile, _ := os.Create(filename)
	// defer targetFile.Close()
	// targetFile.Write(content)

	reader := csv.NewReader(bufio.NewReader(bytes.NewReader(content)))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	records := [][]string{}
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Println("skip wrong row", err)
			continue
		}
		if len(row) < 5 {
			continue
		}
		records = append(records, row)
	}

	TransWechat(records)

	c.JSON(http.StatusOK, gin.H{"message": "Wechat CSV read success"})
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

func SaveImportTransaction(transaction []model.ImportTranscation) error {
	db := core.GetDB()
	for _, tx := range transaction {
		var existing model.ImportTranscation
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
