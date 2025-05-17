package handlers

import (
	"finance-chatbot/api/db"
	"finance-chatbot/api/llm"
	"finance-chatbot/api/models"
	"finance-chatbot/api/mongodb"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type NewConversationRequest struct {
	Message string `json:"message" bson:"message"`
}

type UpdateConversationTitleRequest struct {
	ConversationID string `json:"conversation_id" bson:"conversation_id"`
	Title string `json:"title" bson:"title"`
}

type DeleteConversationRequest struct {
	ConversationID string `json:"conversation_id" bson:"conversation_id"`
}

func HandleCreateNewConversation(c *gin.Context) {

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

	var req NewConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	title, err := llm.GenerateChatTitle(req.Message)

	if err != nil {
		log.Printf("Error generating chat title: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	conversation, err := db.CreateConversation(claims.Sub, title)
	if err != nil {
		log.Printf("Error creating conversation for user %s: %v", claims.Sub, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	conversationContext, err := createConversationContext(c, claims.Sub, conversation.ID.String())
	if err != nil {
		log.Printf("Error creating conversation context for user %s: %v", claims.Sub, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = mongodb.CreateConversationContext(c, conversationContext)
	if err != nil {
		log.Printf("Error saving conversation context to MongoDB for conversation ID %s: %v", conversation.ID.String(), err)

		err = db.DeleteConversation(conversation.ID.String())
		if err != nil {
			log.Printf("Error deleting conversation from DB for conversation ID %s: %v", conversation.ID.String(), err)
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Successfully created new chat for user %s with conversation ID %s", claims.Sub, conversation.ID.String())
	msg := &models.Message{
		ConversationID: conversation.ID.String(),
		Text:           req.Message,
	}
	c.JSON(http.StatusOK, gin.H{"conversation_id": conversation.ID.String(), "conversation_title": conversation.Title})
	processUserMessage(c, claims.Sub, msg)
}

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

func HandleUpdateConversation(c *gin.Context) {
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

	var req UpdateConversationTitleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Error binding JSON for title update: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ConversationID == "" || req.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_id and title are required"})
		return
	}

	// Fetch conversation to check ownership
	conversation, err := db.GetByID(req.ConversationID)
	if err != nil {
		log.Printf("Error fetching conversation: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	if conversation.UserID != claims.Sub {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to update this conversation"})
		return
	}

	updatedConversation, err := db.Update(req.ConversationID, req.Title)
	if err != nil {
		log.Printf("Error updating conversation: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedConversation)
}

func HandleDeleteConversation(c *gin.Context) {
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

	var req DeleteConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Error binding JSON for title update: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ConversationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_id and title are required"})
		return
	}

	// Fetch conversation to check ownership
	conversation, err := db.GetByID(req.ConversationID)
	if err != nil {
		log.Printf("Error fetching conversation: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	if conversation.UserID != claims.Sub {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to update this conversation"})
		return
	}

	err = db.Delete(req.ConversationID)
	if err != nil {
		log.Printf("Error deleting conversation from Postgres: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = mongodb.DeleteConversation(c, req.ConversationID)

	if err != nil {
		log.Printf("Error deleting conversation context from MongoDB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = mongodb.DeleteMessages(c, req.ConversationID)
	if err != nil {
		log.Printf("Error deleting conversation messages from MongoDB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, map[string]any{"success": true})
}