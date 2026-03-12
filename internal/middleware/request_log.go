package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func RequestLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		reqID := c.GetString("requestID")
		latency := time.Since(start)
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		log.Printf("request_id=%s method=%s path=%s status=%d ip=%s latency_ms=%d", reqID, c.Request.Method, path, c.Writer.Status(), c.ClientIP(), latency.Milliseconds())
	}
}
