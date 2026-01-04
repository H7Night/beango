package utils

import (
	"io"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

var LogFile *os.File

var Writer io.Writer = os.Stdout

func InitLogging() (*os.File, error) {
	// 确保日志目录存在
	if err := os.MkdirAll("logs", 0755); err != nil {
		return nil, err
	}
	// 打开日志文件，使用截断模式
	f, err := os.OpenFile("logs/beango.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}

	LogFile = f
	Writer = io.MultiWriter(f, os.Stdout)
	
	// 配置全局日志输出
	log.SetOutput(Writer)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)

	gin.DefaultWriter = Writer
	gin.DefaultErrorWriter = Writer

	log.Println("Log file initialized and truncated: logs/beango.log")

	return f, nil
}
