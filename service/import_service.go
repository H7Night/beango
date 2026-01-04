package service

import (
	"beango/model"
	"beango/utils"
	"bufio"
	"bytes"
	"encoding/csv"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

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
	content, _ := utils.ConvertGBKtoUTF8withBom(baseFile)

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
	if err := TransToBeancount(res, outputFolder, true); err != nil {
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
	if err := TransToBeancount(res, outputFolder, true); err != nil {
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
