package mongodb

import (
	"context"
	"finance-chatbot/api/logger"
	"fmt"
	"os"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
)

var (
	ContextCollection  string = "contexts"
	MessageCollection  string = "messages"
	UserInfoCollection string = "user_info"
	MongoDatabase      string = "conversations"
	MongoClient        *mongo.Client
)

func InitMongoDB() error {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		return fmt.Errorf("MONGO_URI environment variable not set")
	}

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(mongoURI).SetServerAPIOptions(serverAPI)

	client, err := mongo.Connect(opts)
	if err != nil {
		logger.Get().Error("failed to connect to MongoDB",
			zap.String("uri", mongoURI),
			zap.Error(err))
		return fmt.Errorf("error connecting to MongoDB: %v", err)
	}

	MongoClient = client
	logger.Get().Info("successfully connected to MongoDB",
		zap.String("uri", mongoURI))
	return nil
}

func CloseMongoDB() {
	if MongoClient != nil {
		if err := MongoClient.Disconnect(context.TODO()); err != nil {
			logger.Get().Error("failed to disconnect from MongoDB",
				zap.Error(err))
			return
		}
		logger.Get().Info("successfully disconnected from MongoDB")
	}
}
