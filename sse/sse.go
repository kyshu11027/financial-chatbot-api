package sse

import (
	"encoding/json"
	"finance-chatbot/api/models"
	"log"
	"sync"
)

type ClientStream struct {
	Messages chan string
	Done     chan struct{}
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
		log.Printf("No client stream found for conversationID: %s", conversationID)
		return
	}

	var aiResponse models.AIResponse
	if err := json.Unmarshal([]byte(chunk), &aiResponse); err != nil {
		log.Printf("Failed to unmarshal chunk to AIResponse: %v", err)
		return
	}

	log.Printf("AIChunk: %v, LastMessage: %v", aiResponse.Message, aiResponse.LastMessage)

	// If this is the last message, ensure we send the final signal and close channels properly
	if aiResponse.LastMessage {
		select {
		case clientStream.Messages <- "[DONE]":
			log.Printf("Sent [DONE] message to client for conversationID: %s", conversationID)
		default:
			log.Printf("Failed to send [DONE] message: message channel is closed for conversationID: %s", conversationID)
		}

		// Signal completion
		select {
		case clientStream.Done <- struct{}{}:
			log.Printf("Sent done signal to client for conversationID: %s", conversationID)
		default:
			log.Printf("Failed to send done signal: done channel is closed for conversationID: %s", conversationID)
		}

		// Cleanup - you can also close the message channel here if you are done with it
		return
	}

	// Send regular messages
	select {
	case clientStream.Messages <- aiResponse.Message.Message:
		log.Printf("Sent message: %s to client for conversationID: %s", aiResponse.Message.Message, conversationID)
	default:
		log.Printf("Failed to send message: message channel is closed for conversationID: %s", conversationID)
	}
}
