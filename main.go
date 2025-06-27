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
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/plaid/plaid-go/v20/plaid"
	"github.com/stripe/stripe-go/v82"
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

	// Initialize Stripe client
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

}

func main() {
	router := gin.Default()
	// router.SetTrustedProxies([]string{"127.0.0.1", "localhost"}) // May have to update to Cloudflare IPs https://www.cloudflare.com/ips/

	router.Use(middleware.CorsMiddleware)

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
	defer kafka.WorkerPool.Stop()

	// API routes
	api := router.Group("/api")
	{
		api.Use(middleware.AuthMiddleware)
		// Plaid routes
		api.POST("/plaid/link-token/create", handlers.CreateLinkToken)
		api.POST("/plaid/link-token/update", handlers.CreateUpdateLinkToken)
		api.POST("/plaid/token/exchange", handlers.ExchangePublicToken)
		api.POST("/plaid/transaction/list", handlers.GetTransactions)
		api.POST("/plaid/transaction/save", handlers.ProvisionTransactionsJob)
		api.POST("/plaid/account/list", handlers.GetItemsWithAccounts)
		api.POST("/plaid/item/list", handlers.GetItems)
		api.POST("/plaid/item/update", handlers.HandleSuccessfulPlaidItemUpdate)
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
		api.POST("/user/get", handlers.HandleGetUser)
		api.POST("/stripe/session/create", handlers.HandleCreateStripeSession)
	}

	// Webhook routes
	webhook := router.Group("/webhook")
	{
		webhook.POST("/stripe", middleware.StripeWebhookVerifier, handlers.HandleStripeWebhook)
		webhook.POST("/plaid", middleware.PlaidWebhookVerifier, handlers.HandlePlaidWebhook)
	}

	// Public routes
	router.GET("/sse/:conversationID", handlers.HandleSSE)
	router.GET("/metrics", func(c *gin.Context) {
		kafka.WorkerPool.MetricsHandler(c.Writer, c.Request)
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Set up graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Get().Info("Server starting", zap.String("port", port))
		if err := router.Run(":" + port); err != nil {
			logger.Get().Fatal("Failed to start server", zap.Error(err))
		}
	}()

	<-quit
	logger.Get().Info("Shutting down server...")
}
