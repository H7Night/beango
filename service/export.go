package service

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const defaultFolder = "0-default"
const securitFolder = "1-securities"

// TransToBeancount 将交易记录写入 .bean 文件
func TransToBeancount(entries []string, path string) error {
	if len(entries) == 0 {
		return errors.New("trans file is empty")
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

	//如果文件已存在，先删除
	if _, err := os.Stat(path); err == nil {
		if err := os.RemoveAll(path); err != nil {
			log.Printf("failed to remove existing file: %v", err)
			return err
		}
	}

	// 组装default
	for yearMonth, group := range defaultgrouped {
		parts := strings.Split(yearMonth, "-")
		if len(parts) != 2 {
			continue
		}
		year := parts[0]
		month := parts[1]
		dirPath := filepath.Join(path, year, defaultFolder)

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
		dirPath := filepath.Join(path, year, securitFolder)

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
