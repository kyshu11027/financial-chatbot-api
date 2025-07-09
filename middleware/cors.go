package middleware

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func CorsMiddleware(c *gin.Context) {
	switch {
	case strings.HasPrefix(c.Request.URL.Path, "/webhook"):
		// Public webhook: allow any origin, no credentials
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	default:
		// Protected endpoints: restrict origin & allow credentials
		if os.Getenv("ENV") == "production" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", os.Getenv("CLIENT_PROD_URL"))
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Origin", os.Getenv("CLIENT_DEV_URL"))
		}
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
