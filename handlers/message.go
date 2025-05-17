package handlers

import (
	"finance-chatbot/api/logger"
	"finance-chatbot/api/models"
	"finance-chatbot/api/mongodb"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type GetMessagesByConversationIDRequest struct {
	ConversationID string `json:"conversation_id" binding:"required"`
}

func HandleSendMessage(c *gin.Context) {
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

	var req models.Message
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Get().Error("error binding JSON", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.Get().Info("received message request",
		zap.String("conversation_id", req.ConversationID),
		zap.String("user_id", claims.Sub))

	err := processUserMessage(c, claims.Sub, &req)
	if err != nil {
		logger.Get().Error("error processing message",
			zap.String("conversation_id", req.ConversationID),
			zap.String("user_id", claims.Sub),
			zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.Get().Info("message sent successfully",
		zap.String("conversation_id", req.ConversationID),
		zap.String("user_id", claims.Sub))

	c.JSON(http.StatusOK, gin.H{"message": "Message sent successfully"})
}

func HandleGetMessagesByConversationID(c *gin.Context) {
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

	var req GetMessagesByConversationIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Get().Error("error binding JSON", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	messages, err := mongodb.GetMessagesByConversationID(c, claims.Sub, req.ConversationID)
	if err != nil {
		logger.Get().Error("error fetching messages",
			zap.String("conversation_id", req.ConversationID),
			zap.String("user_id", claims.Sub),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(messages) == 0 {
		logger.Get().Info("no messages found",
			zap.String("conversation_id", req.ConversationID),
			zap.String("user_id", claims.Sub))
		c.JSON(http.StatusOK, []models.Message{})
		return
	}

	logger.Get().Info("messages retrieved successfully",
		zap.String("conversation_id", req.ConversationID),
		zap.String("user_id", claims.Sub),
		zap.Int("message_count", len(messages)))

	c.JSON(http.StatusOK, messages)
}
