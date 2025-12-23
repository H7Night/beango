package utils

import (
	"io"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

// LogFile is the opened log file. Close it when the application exits.
var LogFile *os.File

// Writer is the writer used by the application (file + stdout).
var Writer io.Writer = os.Stdout

// InitLogging ensures the logs directory, opens the log file, and configures
// standard log and Gin to write to it. Returns the opened *os.File which
// should be closed by the caller (defer).
func InitLogging() (*os.File, error) {
	if err := os.MkdirAll("logs", 0755); err != nil {
		return nil, err
	}

	f, err := os.OpenFile("logs/beango.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}

	LogFile = f
	Writer = io.MultiWriter(f, os.Stdout)

	// Configure std library logger
	log.SetOutput(Writer)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)

	// Configure Gin to write its logs to the same writer
	gin.DefaultWriter = Writer
	gin.DefaultErrorWriter = Writer

	log.Println("✅ Log file initialized and truncated: logs/beango.log")

	return f, nil
}
