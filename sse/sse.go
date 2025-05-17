package sse

import (
	"encoding/json"
	"finance-chatbot/api/logger"
	"finance-chatbot/api/models"
	"sync"

	"go.uber.org/zap"
)

type ClientStream struct {
	Messages  chan string
	CloseOnce sync.Once
}

var (
	SSEConnections = make(map[string]*ClientStream)
	Mu             sync.RWMutex
)

func SendChunkToClient(conversationID string, chunk string) {
	Mu.RLock()
	clientStream, ok := SSEConnections[conversationID]
	Mu.RUnlock()
	if !ok {
		logger.Get().Info("No client stream found",
			zap.String("conversationID", conversationID))
		return
	}

	var aiResponse models.AIResponse
	if err := json.Unmarshal([]byte(chunk), &aiResponse); err != nil {
		logger.Get().Error("Failed to unmarshal chunk to AIResponse",
			zap.Error(err))
		return
	}

	logger.Get().Debug("Processing AI chunk",
		zap.String("text", aiResponse.Text),
		zap.Bool("lastMessage", aiResponse.LastMessage))

	// If this is the last message, ensure we send the final signal and close channels properly
	if aiResponse.LastMessage && aiResponse.Error {
		clientStream.Messages <- "[ERROR]"
		return
	}

	if aiResponse.LastMessage {
		clientStream.Messages <- "[DONE]"
		return
	}

	// Send regular messages
	select {
	case clientStream.Messages <- aiResponse.Text:
		logger.Get().Debug("Sent message to client",
			zap.String("message", aiResponse.Text),
			zap.String("conversationID", conversationID))
	default:
		logger.Get().Warn("Failed to send message: message channel is closed",
			zap.String("conversationID", conversationID))
	}
}
