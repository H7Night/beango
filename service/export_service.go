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
		return errors.New("import file is empty")
	}

	timeStamp := time.Now().Format("2006-01-02")
	outputDir := filepath.Join(path, timeStamp)

	//如果文件已存在，先删除
	if _, err := os.Stat(outputDir); err == nil {
		if err := os.RemoveAll(outputDir); err != nil {
			fmt.Printf("failed to delete output folder: %v", err)
			return err
		}
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output folder %s: %w", outputDir, err)
	}

	defaultGrouped := make(map[string][]string)
	securityGrouped := make(map[string][]string)

	// 按年月分组
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
		if len(date) < 7 {
			continue
		}
		yearMonth := date[:7]
		// 取出 securities
		if strings.Contains(parts[2], "基金") && strings.Contains(parts[3], "-") {
			securitParts := strings.Split(parts[3], "-")
			if strings.Contains(securitParts[2], "收益发放") {
				securityGrouped[yearMonth] = append(securityGrouped[yearMonth], entry)
				continue
			}
		}
		defaultGrouped[yearMonth] = append(defaultGrouped[yearMonth], entry)
	}
	// 写 default
	if err := writeGroupedEntries(defaultGrouped, outputDir, model.GetConfigString("defaultFolder", "0-default")); err != nil {
		return err
	}

	// 写 securities
	if err := writeGroupedEntries(securityGrouped, outputDir, model.GetConfigString("securitFolder", "1-securities")); err != nil {
		return err
	}
	return nil
}

func writeGroupedEntries(grouped map[string][]string, baseDir, subFolder string) error {
	for yearMonth, group := range grouped {
		parts := strings.Split(yearMonth, "-")
		if len(parts) != 2 {
			continue
		}
		year := parts[0]
		month := parts[1]

		dirPath := filepath.Join(baseDir, year, subFolder)

		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
		}
		beanFile := filepath.Join(dirPath, fmt.Sprintf("%s.bean", month))
		f, err := os.OpenFile(beanFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open bean file: %w", err)
		}

		defer f.Close()

		var builder strings.Builder
		for _, g := range group {
			builder.WriteString(g)
			builder.WriteString("\n")
		}
		if _, err := f.WriteString(builder.String()); err != nil {
			f.Close()
			return fmt.Errorf("failed to write entries: %w", err)
		}
	}
	return nil
}
