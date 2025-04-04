package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

type Message struct {
	Type    string          `json:"type"`
	Content string          `json:"content"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	// Set read deadline to detect stale connections
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			// Send a close message before breaking
			conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			break
		}

		// Reset read deadline after successful read
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		// Here you would:
		// 1. Forward the message to your Python microservice
		// 2. Get the response
		// 3. Send it back to the client

		// For now, we'll just echo back a response
		response := Message{
			Type:    "response",
			Content: "Received your message: " + msg.Content,
		}

		if err := conn.WriteJSON(response); err != nil {
			log.Printf("Failed to write message: %v", err)
			break
		}
	}
} 