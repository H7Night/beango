package utils

import (
	"beango/model"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

var LogFile *os.File
var ConvertLogFile *os.File

var Writer io.Writer = os.Stdout

// logDir 日志目录（由配置 outputFolder 决定，out 已迁移至 test/out）
func logDir() string {
	return model.GetConfigString("outputFolder", model.DefaultOutputFolder)
}

func InitLogging() error {
	// 确保日志目录存在
	if err := os.MkdirAll(logDir(), 0755); err != nil {
		return err
	}
	// 打开日志文件，使用截断模式
	f, err := os.OpenFile(filepath.Join(logDir(), "beango.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	LogFile = f
	Writer = io.MultiWriter(f, os.Stdout)
	
	// 配置全局日志输出
	log.SetOutput(Writer)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)

	gin.DefaultWriter = Writer
	gin.DefaultErrorWriter = Writer

	log.Printf("Log file initialized and truncated: %s", filepath.Join(logDir(), "beango.log"))

	// 初始化转换日志
	cf, err := os.OpenFile(filepath.Join(logDir(), "convert.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	ConvertLogFile = cf

	return nil
}

func CloseLogging() {
	if LogFile != nil {
		LogFile.Close()
	}
	if ConvertLogFile != nil {
		ConvertLogFile.Close()
	}
}

func LogConvert(status string, record interface{}) {
	if ConvertLogFile == nil {
		return
	}
	logger := log.New(ConvertLogFile, "", log.LstdFlags)
	logger.Printf("[%s] %v\n", status, record)
}
