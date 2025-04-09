package handlers

import (
	"finance-chatbot/api/middleware"
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

const LLM_URL = "ws://localhost:8000/chat"

var Connections = make(map[string]*websocket.Conn)

func HandleCreateWebsocketConnection(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	claims, ok := user.(*middleware.SupabaseClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user claims"})
		return
	}

	log.Printf("WebSocket connection established from %s", c.Request.RemoteAddr)
	Connections[claims.Sub] = conn

	// Start a goroutine to monitor the connection
	go monitorConnection(claims.Sub, conn)
}

func HandleCloseWebsocketConnection(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	claims, ok := user.(*middleware.SupabaseClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user claims"})
		return
	}

	Connections[claims.Sub].Close()
	delete(Connections, claims.Sub)
	c.JSON(http.StatusOK, gin.H{"message": "WebSocket connection closed"})
}

func monitorConnection(userID string, conn *websocket.Conn) {
	defer func() {
		conn.Close()
		delete(Connections, userID)
		log.Printf("Connection closed for user %s", userID)
	}()

	for {
		// Set a read deadline
		err := conn.SetReadDeadline(time.Now().Add(60 * time.Second)) // Adjust the timeout as needed
		if err != nil {
			log.Printf("Error setting read deadline: %v", err)
			return
		}

		// Read message from the connection
		_, _, err = conn.ReadMessage()
		if err != nil {
			log.Printf("Connection error for user %s: %v", userID, err)
			return
		}
	}
}
