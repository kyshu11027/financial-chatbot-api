package main

import (
	"finance-chatbot/api/db"
	"finance-chatbot/api/handlers"
	"finance-chatbot/api/middleware"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/plaid/plaid-go/plaid"
)

func init() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Initialize Plaid client
	configuration := plaid.NewConfiguration()
	configuration.AddDefaultHeader("PLAID-CLIENT-ID", os.Getenv("PLAID_CLIENT_ID"))
	configuration.AddDefaultHeader("PLAID-SECRET", os.Getenv("PLAID_SECRET"))
	configuration.UseEnvironment(plaid.Sandbox) // Change to Development or Production as needed
	handlers.PlaidClient = plaid.NewAPIClient(configuration)
}

func main() {
	router := gin.Default()
	router.SetTrustedProxies([]string{"127.0.0.1", "localhost"}) // Only trust local proxies

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Initialize database
	db.InitDB()
	// Authentication middleware
	router.Use(middleware.AuthMiddleware)

	// API routes
	api := router.Group("/api")
	{
		// Plaid routes
		api.POST("/plaid/create-link-token", handlers.CreateLinkToken)
		api.POST("/plaid/exchange-token", handlers.ExchangePublicToken)
		api.POST("/plaid/transactions", handlers.GetTransactions)
		api.POST("/plaid/items", handlers.GetItems)

		// WebSocket route
		api.GET("/ws", handlers.HandleWebSocket)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
} 