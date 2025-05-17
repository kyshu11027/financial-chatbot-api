package handlers

import (
	"encoding/json"
	"finance-chatbot/api/logger"
	"finance-chatbot/api/sse"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type SSEMessage struct {
	Message string `json:"message"`
}

func HandleSSE(c *gin.Context) {
	if err := authenticateSSE(c); err != nil {
		logger.Get().Error("authentication failed", zap.Error(err))
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

	logger.Get().Info("SSE connection established",
		zap.String("conversation_id", conversationID))

	// Automatically remove connection when client disconnects
	defer func() {
		logger.Get().Info("closing SSE connection",
			zap.String("conversation_id", conversationID))
		sse.Mu.Lock()
		delete(sse.SSEConnections, conversationID)
		sse.Mu.Unlock()
		logger.Get().Info("SSE connection closed",
			zap.String("conversation_id", conversationID))
	}()

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		logger.Get().Error("streaming not supported")
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
				logger.Get().Error("failed to marshal SSE message",
					zap.Error(err),
					zap.String("message", msg))
				return false
			}

			c.Writer.Write([]byte("data: " + string(payload) + "\n\n"))
			flusher.Flush()
			return true
		case <-c.Request.Context().Done():
			logger.Get().Info("SSE context done",
				zap.String("conversation_id", conversationID),
				zap.Error(c.Request.Context().Err()))
			return false
		}
	})
}
