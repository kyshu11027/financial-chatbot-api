package handlers

import (
	"context"
	"encoding/json"
	"finance-chatbot/api/kafka"
	"finance-chatbot/api/middleware"
	"finance-chatbot/api/models"
	"finance-chatbot/api/mongodb"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type MessageRequest struct {
	models.Message
}

type AIResponse struct {
	models.Message
}

func HandleSendMessage(c *gin.Context) {

	user, exists := c.Get("user")
	if !exists {
		log.Println("User not authenticated")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	claims, ok := user.(*middleware.SupabaseClaims)
	if !ok {
		log.Println("Invalid user claims")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user claims"})
		return
	}

	var req MessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	log.Printf("Received message request: %+v", req)
	req.UserID = claims.Sub
	req.Sender = "UserMessage"
	
	err := mongodb.CreateMessage(context.Background(), &req.Message)
	if err != nil {
		log.Printf("Failed to create message: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create message"})
		return
	}

	// Marshal the request and handle the error
	messageBytes, err := json.Marshal(req)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal message"})
		return
	}
	err = kafka.ProduceMessage(kafka.MessageTopic, messageBytes)
	if err != nil {
		log.Printf("Failed to produce message: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to produce message"})
		return
	}
	log.Println("Message sent successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Message sent successfully"})
}

func HandleReceiveMessage(c *gin.Context) {
	var req AIResponse
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	conn := Connections[req.UserID]
	if conn == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Connection not found"})
		return
	}

	// Marshal the message to JSON
	messageBytes, err := json.Marshal(req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal message"})
		return
	}

	// Send the message over the WebSocket connection
	err = conn.WriteMessage(websocket.TextMessage, messageBytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message sent successfully"})
}
