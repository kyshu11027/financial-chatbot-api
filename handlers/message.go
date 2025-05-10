package handlers

import (
	"finance-chatbot/api/models"
	"finance-chatbot/api/mongodb"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type GetMessagesByConversationIDRequest struct {
	ConversationID string `json:"conversation_id" binding:"required"`
}

func HandleSendMessage(c *gin.Context) {

	user, exists := c.Get("user")
	if !exists {
		log.Println("User not authenticated")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	claims, ok := user.(*models.SupabaseClaims)
	if !ok {
		log.Println("Invalid user claims")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user claims"})
		return
	}

	var req models.Message
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	log.Printf("Received message request: %+v", req)
	err := processUserMessage(c, claims.Sub, &req)
	if err != nil {
		log.Printf("Error processing message: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	log.Println("Message sent successfully")

	c.JSON(http.StatusOK, gin.H{"message": "Message sent successfully"})
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
