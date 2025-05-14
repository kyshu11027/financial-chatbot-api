package handlers

import (
	"encoding/json"
	"finance-chatbot/api/sse"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type SSEMessage struct {
	Message string `json:"message"`
}

func HandleSSE(c *gin.Context) {
	if err := authenticateSSE(c); err != nil {
		log.Printf("Authentication failed: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("Unauthorized: %v", err)})
		return
	}

	conversationID := c.Param("conversationID")

	messageChan := make(chan string, 100)
	// doneChan := make(chan struct{})

	clientStream := &sse.ClientStream{
		Messages: messageChan,
		// Done:     doneChan,
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
			payload, err := json.Marshal(SSEMessage{Message: msg})
			if err != nil {
				log.Println("failed to marshal SSE message:", err)
				return false
			}

			c.Writer.Write([]byte("data: " + string(payload) + "\n\n"))
			flusher.Flush()
			return true
		case <-c.Request.Context().Done():
			log.Println("context done:", c.Request.Context().Err())
			return false
		}
	})
}
