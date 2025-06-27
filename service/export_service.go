package service

import (
	"beango/model"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TransToBeancount 将交易记录写入 .bean 文件
func TransToBeancount(entries []string, path string) error {
	if len(entries) == 0 {
		return errors.New("导入文件内容为空")
	}

	today := time.Now().Format("2006-01-02")
	timelyDir := filepath.Join(path, today)

	//如果文件已存在，先删除
	if _, err := os.Stat(timelyDir); err == nil {
		if err := os.RemoveAll(timelyDir); err != nil {
			fmt.Printf("删除目录失败: %v", err)
			return err
		}
	}

	if err := os.MkdirAll(timelyDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败 %s: %w", timelyDir, err)
	}

	defaultgrouped := make(map[string][]string)
	securitGrouped := make(map[string][]string)
	for _, entry := range entries {
		lines := strings.Split(entry, "\n")
		if len(lines) == 0 {
			continue
		}
		parts := strings.Split(lines[0], " ")
		if len(parts) < 1 {
			continue
		}
		date := parts[0]
		yearMonth := date[:7]
		// 取出 securities
		if strings.Contains(parts[2], "基金") && strings.Contains(parts[3], "-") {
			securitParts := strings.Split(parts[3], "-")
			if strings.Contains(securitParts[2], "收益发放") {
				securitGrouped[yearMonth] = append(securitGrouped[yearMonth], entry)
				continue
			}
		}
		defaultgrouped[yearMonth] = append(defaultgrouped[yearMonth], entry)
	}

	// 组装default
	for yearMonth, group := range defaultgrouped {
		parts := strings.Split(yearMonth, "-")
		if len(parts) != 2 {
			continue
		}
		year := parts[0]
		month := parts[1]

		var defaultFolder = model.GetConfigString("defaultFolder", "0-default")
		dirPath := filepath.Join(timelyDir, year, defaultFolder)

		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
		}
		beanFile := filepath.Join(dirPath, fmt.Sprintf("%s.bean", month))
		f, err := os.OpenFile(beanFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open bean file: %w", err)
		}
		for _, g := range group {
			if _, err := f.WriteString(g + "\n"); err != nil {
				f.Close()
				return fmt.Errorf("failed to write entry: %w", err)
			}
		}
	}

	//组装 securities
	for yearMonth, group := range securitGrouped {
		parts := strings.Split(yearMonth, "-")
		if len(parts) != 2 {
			continue
		}
		year := parts[0]
		month := parts[1]

		securitFolder := model.GetConfigString("securitFolder", "1-securities")
		dirPath := filepath.Join(timelyDir, year, securitFolder)

		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
		}
		beanFile := filepath.Join(dirPath, fmt.Sprintf("%s.bean", month))
		f, err := os.OpenFile(beanFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open bean file: %w", err)
		}
		for _, g := range group {
			if _, err := f.WriteString(g + "\n"); err != nil {
				f.Close()
				return fmt.Errorf("failed to write entry: %w", err)
			}
		}
	}
	return nil
}
