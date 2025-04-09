package handlers

import (
	"encoding/json"
	"finance-chatbot/api/middleware"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)


type ClientMessage struct {
	ConversationID string          `json:"conversation_id"`
	Message        json.RawMessage `json:"message,omitempty"`
}

type LLMResponse struct {
	ConversationID string          `json:"conversation_id"`
	Message        json.RawMessage `json:"message,omitempty"`
	Type           string          `json:"type"`
}
func HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	user, exists := c.Get("user")
	if !exists {
		log.Printf("User not authenticated")
		return
	}

	claims, ok := user.(*middleware.SupabaseClaims)
	if !ok {
		log.Printf("Invalid user claims")
		return
	}

	Connections[claims.Sub] = conn

	log.Printf("WebSocket connection established from %s", c.Request.RemoteAddr)

	// Set read deadline to detect stale connections
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	for {
		var msg ClientMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			} else if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				log.Printf("Client closed connection normally")
			} else {
				log.Printf("Error reading message: %v", err)
			}
			// Send a close message before breaking
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			break
		}

		log.Printf("Received message: %+v", msg)

		// Reset read deadline after successful read
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		// Connect to Python LLM websocket service
		llmConn, _, err := websocket.DefaultDialer.Dial(LLM_URL, nil)
		if err != nil {
			log.Printf("Failed to connect to LLM service: %v", err)
			break
		}
		defer llmConn.Close()

		// Set read deadline for LLM connection
		llmConn.SetReadDeadline(time.Now().Add(30 * time.Second))

		// Forward client message to LLM service
		if err := llmConn.WriteJSON(msg); err != nil {
			log.Printf("Failed to send message to LLM service: %v", err)
			break
		}

		// Read streaming responses from LLM service and forward to client
		for {
			var llmResponse LLMResponse
			err := llmConn.ReadJSON(&llmResponse)
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					break // LLM service finished streaming
				}
				log.Printf("Failed to read LLM response: %v", err)
				break
			}

			// Forward each chunk to the client
			if err := conn.WriteJSON(llmResponse); err != nil {
				log.Printf("Failed to write LLM response to client: %v", err)
				break
			}

			// If this is the final message, break the streaming loop
			if llmResponse.Type == "end" {
				log.Printf("LLM service finished streaming")
				llmConn.Close()
				break
			}
		}
	}
	log.Printf("WebSocket connection closed for %s", c.Request.RemoteAddr)
}