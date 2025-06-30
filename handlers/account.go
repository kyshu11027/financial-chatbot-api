package handlers

import (
	"finance-chatbot/api/db"
	"finance-chatbot/api/logger"
	"finance-chatbot/api/models"
	"finance-chatbot/api/mongodb"
	"finance-chatbot/api/qdrant"
	"net/http"

	"github.com/gin-gonic/gin"
)

func HandleDeleteAccount(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		logger.Get().Error("user not authenticated")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	claims, ok := user.(*models.SupabaseClaims)
	if !ok {
		logger.Get().Error("invalid user claims")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user claims"})
		return
	}

	err := mongodb.DeleteContextsByUserID(c, claims.Sub)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting user conversation contexts"})
	}

	err = mongodb.DeleteUserInfo(c, claims.Sub)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting user info"})
	}

	err = mongodb.DeleteMessagesByUserID(c, claims.Sub)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting conversation messages"})
	}

	err = db.DeleteConversationsByUserID(claims.Sub)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting conversations"})
	}

	err = qdrant.DeleteTransactionsByUserID(claims.Sub)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting transactions from Qdrant"})
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
