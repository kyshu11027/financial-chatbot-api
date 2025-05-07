package handlers

import (
	"context"
	"encoding/json"
	"finance-chatbot/api/kafka"
	"finance-chatbot/api/models"
	"finance-chatbot/api/mongodb"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

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

func processUserMessage(ctx context.Context, userId string, msg *models.Message) error {
	msg.UserID = userId
	msg.Sender = "UserMessage"
	msg.Timestamp = time.Now().Unix()

	err := mongodb.CreateMessage(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	messageBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = kafka.ProduceMessage(kafka.MessageTopic, messageBytes)
	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	return nil
}
