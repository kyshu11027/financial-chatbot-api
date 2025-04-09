package mongodb

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ContextCollection string = "contexts"
var MessageCollection string = "messages"
var MongoDatabase string = "conversations"
var MongoClient *mongo.Client

func InitMongoDB() error {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		return fmt.Errorf("MONGO_URI environment variable not set")
	}

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(mongoURI).SetServerAPIOptions(serverAPI)

	client, err := mongo.Connect(opts)
	if err != nil {
		return fmt.Errorf("error connecting to MongoDB: %v", err)
	}

	MongoClient = client
	log.Println("Successfully connected to MongoDB")
	return nil
}

func CloseMongoDB() {
	if MongoClient != nil {
		MongoClient.Disconnect(context.TODO())
	}
}
