package handlers

import (
	"finance-chatbot/api/db"
	"finance-chatbot/api/models"
	"finance-chatbot/api/mongodb"
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

	claims, ok := user.(*models.SupabaseClaims)
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

type GetMessagesByConversationIDRequest struct {
	ConversationID string `json:"conversation_id" binding:"required"`
}

func HandleGetMessagesByConversationID(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	claims, ok := user.(*models.SupabaseClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user claims"})
		return
	}

	var req GetMessagesByConversationIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	messages, err := mongodb.GetMessagesByConversationID(c, claims.Sub, req.ConversationID)

	if err != nil {
		log.Printf("Error fetching messages for conversation %s: %v", req.ConversationID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(messages) == 0 {
		log.Printf("No messages found for conversation %s", req.ConversationID)
		c.JSON(http.StatusOK, []models.Message{})
		return
	}
	c.JSON(http.StatusOK, messages)
}
