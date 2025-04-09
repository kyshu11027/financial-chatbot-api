package middleware

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func MicroserviceAuthMiddleware(c *gin.Context) {
	apiKey := c.Request.Header.Get("X-API-Key")
	if apiKey != os.Getenv("INTERNAL_API_KEY") { // Replace with your actual API key
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}
	c.Next()
}
