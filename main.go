package main

import (
	"finance-chatbot/api/db"
	"finance-chatbot/api/handlers"
	"finance-chatbot/api/kafka"
	"finance-chatbot/api/logger"
	"finance-chatbot/api/middleware"
	"finance-chatbot/api/mongodb"
	"flag"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/plaid/plaid-go/plaid"
	"go.uber.org/zap"
)

func init() {
	// Define command line flags
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	// Initialize logger first
	if err := logger.Init(os.Getenv("ENV") == "development", logger.LogLevel(*logLevel)); err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	// Load environment variables after logger is initialized
	if err := godotenv.Load(); err != nil {
		logger.Get().Error("Warning: .env file not found")
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

	// Initialize databases
	if err := db.InitDB(); err != nil {
		logger.Get().Fatal("Failed to initialize database", zap.Error(err))
	}
	defer db.CloseDB()

	if err := mongodb.InitMongoDB(); err != nil {
		logger.Get().Fatal("Failed to initialize MongoDB", zap.Error(err))
	}
	defer mongodb.CloseMongoDB()

	if err := kafka.InitProducer(); err != nil {
		logger.Get().Fatal("Failed to initialize Kafka producer", zap.Error(err))
	}
	defer kafka.MessageProducer.Close()

	err := kafka.StartKafkaConsumer()
	if err != nil {
		logger.Get().Fatal("Failed to start Kafka consumer", zap.Error(err))
	}

	// API routes
	api := router.Group("/api")
	{
		api.Use(middleware.AuthMiddleware)
		// Plaid routes
		api.POST("/plaid/link-token/create", handlers.CreateLinkToken)
		api.POST("/plaid/token/exchange", handlers.ExchangePublicToken)
		api.POST("/plaid/transaction/list", handlers.GetTransactions)
		api.POST("/plaid/item/list", handlers.GetItems)
		api.POST("/chat/conversation/new", handlers.HandleCreateNewConversation)
		api.POST("/chat/conversation/list", handlers.HandleGetConversations)
		api.POST("/chat/conversation/update", handlers.HandleUpdateConversation)
		api.POST("/chat/conversation/delete", handlers.HandleDeleteConversation)
		api.POST("/chat/message/list", handlers.HandleGetMessagesByConversationID)
		api.POST("/chat/message/send", handlers.HandleSendMessage)
		api.POST("/user-info/create", handlers.CreateUserInfo)
		api.POST("/user-info/update", handlers.UpdateUserInfo)
		api.POST("/user-info/delete", handlers.DeleteUserInfo)
		api.POST("/user-info/get", handlers.GetUserInfo)
	}
	router.GET("/sse/:conversationID", handlers.HandleSSE)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Get().Info("Server starting", zap.String("port", port))
	if err := router.Run(":" + port); err != nil {
		logger.Get().Fatal("Failed to start server", zap.Error(err))
	}
}
