package handlers

import (
	"finance-chatbot/api/models"
	"finance-chatbot/api/sse"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func HandleSSE(c *gin.Context) {
	if err := authenticateSSE(c); err != nil {
		log.Printf("Authentication failed: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("Unauthorized: %v", err)})
		return
	}

	conversationID := c.Param("conversationID")

	messageChan := make(chan string, 100)
	doneChan := make(chan struct{})

	clientStream := &sse.ClientStream{
		Messages: messageChan,
		Done:     doneChan,
	}

	sse.Mu.Lock()
	sse.SSEConnections[conversationID] = clientStream
	sse.Mu.Unlock()

	log.Printf("SSE connection established for conversationID: %s", conversationID)

	// Automatically remove connection when client disconnects
	defer func() {
		log.Printf("Closing SSE connection for conversationID: %s", conversationID)
		sse.Mu.Lock()
		delete(sse.SSEConnections, conversationID)
		sse.Mu.Unlock()
		log.Printf("SSE connection closed for conversationID: %s", conversationID)
	}()

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.String(http.StatusInternalServerError, "Streaming unsupported!")
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	c.Stream(func(w io.Writer) bool {
		select {
		case msg, ok := <-messageChan:
			if !ok {
				return false
			}
			c.Writer.Write([]byte("data: " + msg + "\n\n"))
			flusher.Flush()
			return true
		case <-c.Request.Context().Done():
			log.Println("context done:", c.Request.Context().Err())
			return false
		case <-doneChan:
			log.Printf("Done signal received for conversationID: %s", conversationID)
			return false
		}
	})
}

func authenticateSSE(c *gin.Context) error{
	tokenString := c.DefaultQuery("token", "")
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid token"})
		c.Abort()
		return fmt.Errorf("missing or invalid token")
	}

	claims := &models.SupabaseClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method is HS256
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		// Use the JWT secret for verification
		secret := os.Getenv("SUPABASE_JWT_SECRET")
		if secret == "" {
			return nil, fmt.Errorf("SUPABASE_JWT_SECRET environment variable not set")
		}
		return []byte(secret), nil
	})

	if err != nil {
		log.Printf("Error parsing claims: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
		c.Abort()
		return err
	}

	if !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		c.Abort()
		return err
	}

	// Verify issuer
	if claims.Issuer != os.Getenv("SUPABASE_URL")+"/auth/v1" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token issuer"})
		c.Abort()
		return err
	}
	return nil
}