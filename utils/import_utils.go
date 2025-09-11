package utils

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// UTF-8 BOM 的字节序列
var utf8Bom = []byte{0xEF, 0xBB, 0xBF}

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
