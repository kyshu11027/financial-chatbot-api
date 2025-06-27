package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func CorsMiddleware(c *gin.Context) {
	switch {
	case strings.HasPrefix(c.Request.URL.Path, "/webhook"):
		// Public webhook: allow any origin, no credentials
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	case strings.HasPrefix(c.Request.URL.Path, "/sse"):
		// SSE endpoint — often public, allow any origin
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
	default:
		// Protected endpoints: restrict origin & allow credentials
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000") // or hardcode frontend URL
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

	if c.Request.Method == "OPTIONS" {
		c.AbortWithStatus(204)
		return
	}

	c.Next()
}
