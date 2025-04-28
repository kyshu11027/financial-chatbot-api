package handlers

import (
	"finance-chatbot/api/sse"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func HandleSSE(c *gin.Context) {
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
		close(messageChan)
		close(doneChan)
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
			c.Writer.Write([]byte("data:" + msg + "\n\n"))
			flusher.Flush()
			return true
		case <-c.Request.Context().Done():
			return false
		case <-doneChan:
			return false
		}
	})
}
