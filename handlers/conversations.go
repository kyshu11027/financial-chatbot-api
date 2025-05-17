package handlers

import (
	"finance-chatbot/api/db"
	"finance-chatbot/api/llm"
	"finance-chatbot/api/logger"
	"finance-chatbot/api/models"
	"finance-chatbot/api/mongodb"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type NewConversationRequest struct {
	Message string `json:"message" bson:"message"`
}

type UpdateConversationTitleRequest struct {
	ConversationID string `json:"conversation_id" bson:"conversation_id"`
	Title          string `json:"title" bson:"title"`
}

type DeleteConversationRequest struct {
	ConversationID string `json:"conversation_id" bson:"conversation_id"`
}

func HandleCreateNewConversation(c *gin.Context) {
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

	var req NewConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Get().Error("error binding JSON", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	title, err := llm.GenerateChatTitle(req.Message)
	if err != nil {
		logger.Get().Error("error generating chat title", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	conversation, err := db.CreateConversation(claims.Sub, title)
	if err != nil {
		logger.Get().Error("error creating conversation",
			zap.String("user_id", claims.Sub),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	conversationContext, err := createConversationContext(c, claims.Sub, conversation.ID.String())
	if err != nil {
		logger.Get().Error("error creating conversation context",
			zap.String("user_id", claims.Sub),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = mongodb.CreateConversationContext(c, conversationContext)
	if err != nil {
		logger.Get().Error("error saving conversation context to MongoDB",
			zap.String("conversation_id", conversation.ID.String()),
			zap.Error(err))

		err = db.DeleteConversation(conversation.ID.String())
		if err != nil {
			logger.Get().Error("error deleting conversation from DB",
				zap.String("conversation_id", conversation.ID.String()),
				zap.Error(err))
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Get().Info("successfully created new chat",
		zap.String("user_id", claims.Sub),
		zap.String("conversation_id", conversation.ID.String()))

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

	conversations, err := db.GetAllByUserID(claims.Sub)
	if err != nil {
		logger.Get().Error("error fetching conversations",
			zap.String("user_id", claims.Sub),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, conversations)
}

func HandleUpdateConversation(c *gin.Context) {
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

	var req UpdateConversationTitleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Get().Error("error binding JSON for title update", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ConversationID == "" || req.Title == "" {
		logger.Get().Error("missing required fields",
			zap.String("conversation_id", req.ConversationID),
			zap.String("title", req.Title))
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_id and title are required"})
		return
	}

	conversation, err := db.GetByID(req.ConversationID)
	if err != nil {
		logger.Get().Error("error fetching conversation", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	if conversation.UserID != claims.Sub {
		logger.Get().Error("unauthorized conversation update attempt",
			zap.String("user_id", claims.Sub),
			zap.String("conversation_id", req.ConversationID))
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to update this conversation"})
		return
	}

	updatedConversation, err := db.Update(req.ConversationID, req.Title)
	if err != nil {
		logger.Get().Error("error updating conversation",
			zap.String("conversation_id", req.ConversationID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Get().Info("conversation updated successfully",
		zap.String("conversation_id", req.ConversationID),
		zap.String("new_title", req.Title))
	c.JSON(http.StatusOK, updatedConversation)
}

func HandleDeleteConversation(c *gin.Context) {
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

	var req DeleteConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Get().Error("error binding JSON for deletion", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ConversationID == "" {
		logger.Get().Error("missing conversation_id")
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_id is required"})
		return
	}

	conversation, err := db.GetByID(req.ConversationID)
	if err != nil {
		logger.Get().Error("error fetching conversation", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	if conversation.UserID != claims.Sub {
		logger.Get().Error("unauthorized conversation deletion attempt",
			zap.String("user_id", claims.Sub),
			zap.String("conversation_id", req.ConversationID))
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to delete this conversation"})
		return
	}

	err = db.Delete(req.ConversationID)
	if err != nil {
		logger.Get().Error("error deleting conversation from Postgres",
			zap.String("conversation_id", req.ConversationID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = mongodb.DeleteConversation(c, req.ConversationID)
	if err != nil {
		logger.Get().Error("error deleting conversation context from MongoDB",
			zap.String("conversation_id", req.ConversationID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = mongodb.DeleteMessages(c, req.ConversationID)
	if err != nil {
		logger.Get().Error("error deleting conversation messages from MongoDB",
			zap.String("conversation_id", req.ConversationID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Get().Info("conversation deleted successfully",
		zap.String("conversation_id", req.ConversationID))
	c.JSON(http.StatusOK, map[string]any{"success": true})
}
