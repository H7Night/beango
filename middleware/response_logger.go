package middleware

import (
	"bytes"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// bodyWriter captures response body written by handlers
type bodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// ResponseLoggingMiddleware logs response details including status and body
// right after the handler runs so you can see the program message returned by c.JSON.
func ResponseLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		bw := &bodyWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = bw
		start := time.Now()
		c.Next()
		latency := time.Since(start)
		status := c.Writer.Status()
		// Limit body size logged to avoid excessive output
		bodyStr := bw.body.String()
		if len(bodyStr) > 4096 {
			bodyStr = bodyStr[:4096] + "... (truncated)"
		}
		log.Printf("[%s] %s %s %d %s body=%s latency=%s\n",
			start.Format("2006-01-02 15:04:05"),
			c.ClientIP(),
			c.Request.Method,
			status,
			c.Request.URL.Path,
			bodyStr,
			latency.String(),
		)
	}
}