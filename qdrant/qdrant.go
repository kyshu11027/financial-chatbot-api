package qdrant

import (
	"fmt"
	"os"

	"finance-chatbot/api/logger"

	"github.com/qdrant/go-client/qdrant"
	"go.uber.org/zap"
)

var (
	QdrantClient           *qdrant.Client
	TransactionsCollection = "transactions"
)

// InitQdrantClient initializes the Qdrant Cloud client.
func InitQdrantClient() error {
	host := os.Getenv("QDRANT_URL")
	if host == "" {
		return fmt.Errorf("QDRANT_HOST environment variable not set")
	}

	apiKey := os.Getenv("QDRANT_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("QDRANT_API_KEY environment variable not set")
	}

	port := 6334 // Default secure gRPC port for Qdrant Cloud

	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   host,
		Port:   port,
		APIKey: apiKey,
		UseTLS: true,
	})
	if err != nil {
		logger.Get().Error("failed to connect to Qdrant Cloud",
			zap.String("host", host),
			zap.Error(err))
		return fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	QdrantClient = client
	logger.Get().Info("successfully connected to Qdrant Cloud",
		zap.String("host", host))

	return nil
}

// CloseQdrantClient closes the Qdrant connection (if needed).
func CloseQdrantClient() {
	// Currently, qdrant.Client does not expose a Close() method,
	// but this function is here for future compatibility.
	QdrantClient = nil
	logger.Get().Info("Qdrant client cleaned up")
}
