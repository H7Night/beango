package service

import (
	"beango/model"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// TransToBeancount 将交易记录写入 .bean 文件
func TransToBeancount(entries []string, path string, isMerge bool) error {
	if len(entries) == 0 {
		return errors.New("import file is empty")
	}

	var outputDir string
	if isMerge {
		outputDir = path
	} else {
		timeStamp := time.Now().Format("2006-01-02")
		outputDir = filepath.Join(path, timeStamp)

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
	if err := writeGroupedEntries(defaultGrouped, outputDir, model.GetConfigString("defaultFolder", "0-default"), isMerge); err != nil {
		return err
	}

	// 写 securities
	if err := writeGroupedEntries(securityGrouped, outputDir, model.GetConfigString("securitFolder", "1-securities"), isMerge); err != nil {
		return err
	}
	return nil
}

func writeGroupedEntries(grouped map[string][]string, baseDir, subFolder string, isMerge bool) error {
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

		var allEntries []string
		if isMerge {
			if existing, err := parseBeanFile(beanFile); err == nil {
				allEntries = append(allEntries, existing...)
			}
		}
		allEntries = append(allEntries, group...)

		// 按时间排序
		sort.Slice(allEntries, func(i, j int) bool {
			ti, err := getEntryTime(allEntries[i])
			if err != nil {
				return false
			}
			tj, err := getEntryTime(allEntries[j])
			if err != nil {
				return false
			}
			return ti.After(tj)
		})

		f, err := os.OpenFile(beanFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open bean file: %w", err)
		}

		defer f.Close()

		var builder strings.Builder
		for _, e := range allEntries {
			builder.WriteString(e)
			builder.WriteString("\n\n")
		}
		if _, err := f.WriteString(builder.String()); err != nil {
			f.Close()
			return fmt.Errorf("failed to write entries: %w", err)
		}
	}
	return nil
}

func getEntryTime(entry string) (time.Time, error) {
	lines := strings.Split(entry, "\n")
	if len(lines) == 0 {
		return time.Time{}, errors.New("empty entry")
	}
	dateStr := strings.Split(lines[0], " ")[0]
	for _, line := range lines {
		if strings.HasPrefix(line, "    time: ") {
			timeStr := strings.Trim(strings.TrimPrefix(line, "    time: "), "\"")
			datetimeStr := dateStr + " " + timeStr
			return time.Parse("2006-01-02 15:04:05", datetimeStr)
		}
	}
	return time.Time{}, errors.New("no time found")
}

func parseBeanFile(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	entries := strings.Split(string(content), "\n\n")
	var res []string
	for _, e := range entries {
		e = strings.TrimSpace(e)
		if e != "" {
			res = append(res, e)
		}
	}
	return res, nil
}
