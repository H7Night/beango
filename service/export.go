package service

import (
	"errors"
	"log"
	"os"
)

// TransToBeancount 将交易记录写入 .bean 文件
func TransToBeancount(entries []string, path string) error {
	if len(entries) == 0 {
		return errors.New("trans file is empty")
	}
	// 如果文件已存在，先删除
	if _, err := os.Stat(path); err == nil {
		if err := os.Remove(path); err != nil {
			log.Printf("failed to remove existing file: %v", err)
			return err
		}
	}

	file, err := os.Create(path)
	if err != nil {
		log.Printf("failed to create file: %v", err)
		return err
	}
	defer file.Close()

	for _, entry := range entries {
		if _, err := file.WriteString(entry + "\n\n"); err != nil {
			log.Println("failed to write entry: %v", err)
			return err
		}
	}

	return nil
}
