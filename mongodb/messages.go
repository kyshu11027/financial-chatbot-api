package mongodb

import (
	"context"
	"finance-chatbot/api/models"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func CreateMessage(ctx context.Context, message *models.Message) error {
	collection := MongoClient.Database(MongoDatabase).Collection(MessageCollection)
	_, err := collection.InsertOne(ctx, message)
	if err != nil {
		return fmt.Errorf("error creating mongo item: %v", err)
	}
	return nil
}

func GetMessagesByConversationID(ctx context.Context, userID string, conversationID string) ([]models.Message, error) {
	collection := MongoClient.Database(MongoDatabase).Collection(MessageCollection)
	filter := bson.M{
		"conversation_id": conversationID,
	}
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}})

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("error fetching messages: %v", err)
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	for cursor.Next(ctx) {
		var message models.Message
		if err := cursor.Decode(&message); err != nil {
			return nil, fmt.Errorf("error decoding message: %v", err)
		}
		messages = append(messages, message)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %v", err)
	}

	return messages, nil
}

func DeleteMessages(ctx context.Context, conversationID string) error {
	collection := MongoClient.Database(MongoDatabase).Collection(MessageCollection)
	_, err := collection.DeleteMany(ctx, map[string]string{"conversation_id": conversationID})
	if err != nil {
		return fmt.Errorf("error deleting messages: %v", err)
	}
	return nil
}
