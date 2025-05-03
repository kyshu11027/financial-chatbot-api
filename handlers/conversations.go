package handlers

import (
	"finance-chatbot/api/db"
	"finance-chatbot/api/middleware"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func HandleGetConversations(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	claims, ok := user.(*middleware.SupabaseClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user claims"})
		return
	}

	conversations, err := db.GetAllByUserID(claims.Sub)
	if err != nil {
		log.Printf("Error fetching conversations for user %s: %v", claims.Sub, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, conversations)
}
