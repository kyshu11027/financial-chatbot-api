package sse

import (
	"encoding/json"
	"finance-chatbot/api/logger"
	"finance-chatbot/api/models"
	"sync"

	"go.uber.org/zap"
)

type ClientStream struct {
	Messages      chan string
	BufferFlushed chan struct{} // closed once buffer is flushed
	CloseOnce     sync.Once
}

var (
	SSEConnections = make(map[string]*ClientStream)
	BufferedChunks = make(map[string][]string)
	Mu             sync.RWMutex
)

// RegisterClient adds a client and flushes any buffered messages
func RegisterClient(conversationID string, stream *ClientStream) {
	Mu.Lock()
	SSEConnections[conversationID] = stream
	buffer := BufferedChunks[conversationID]
	delete(BufferedChunks, conversationID)
	Mu.Unlock()

	go func() {
		for _, chunk := range buffer {
			var aiResponse models.AIResponse
			if err := json.Unmarshal([]byte(chunk), &aiResponse); err == nil {
				message := resolveMessage(aiResponse)
				stream.Messages <- message
			} else {
				logger.Get().Error("Failed to unmarshal buffered chunk",
					zap.Error(err),
					zap.String("conversationID", conversationID))
			}
		}
		close(stream.BufferFlushed) // signal that flushing is complete
	}()
}

// SendChunkToClient sends a message to the client's stream, preserving order
func SendChunkToClient(conversationID string, chunk string) {
	var aiResponse models.AIResponse
	if err := json.Unmarshal([]byte(chunk), &aiResponse); err != nil {
		logger.Get().Error("Failed to unmarshal chunk to AIResponse",
			zap.Error(err))
		return
	}

	Mu.RLock()
	clientStream, ok := SSEConnections[conversationID]
	Mu.RUnlock()

	if !ok {
		// No client, but last message received. Delete the buffer.
		if aiResponse.LastMessage {
			Mu.Lock()
			delete(BufferedChunks, conversationID)
			Mu.Unlock()
			logger.Get().Info("Deleted conversation buffer because last message received",
				zap.String("conversationID", conversationID))
			return
		}
		// No client yet: buffer the chunk
		Mu.Lock()
		BufferedChunks[conversationID] = append(BufferedChunks[conversationID], chunk)
		Mu.Unlock()

		logger.Get().Info("Buffered chunk because client not connected yet",
			zap.String("conversationID", conversationID))
		return
	}

	// Wait for buffer flush if still flushing
	<-clientStream.BufferFlushed

	message := resolveMessage(aiResponse)

	select {
	case clientStream.Messages <- message:
		logger.Get().Debug("Sent message to client",
			zap.String("message", message),
			zap.String("conversationID", conversationID))
	default:
		logger.Get().Warn("Client message channel is blocked",
			zap.String("conversationID", conversationID))
	}
}

// UnregisterClient cleans up client resources
func UnregisterClient(conversationID string) {
	Mu.Lock()
	delete(SSEConnections, conversationID)
	delete(BufferedChunks, conversationID)
	Mu.Unlock()
}

// resolveMessage converts AIResponse into string message
func resolveMessage(resp models.AIResponse) string {
	switch {
	case resp.LastMessage && resp.Error:
		return "[ERROR]"
	case resp.LastMessage:
		return "[DONE]"
	default:
		return resp.Text
	}
}
