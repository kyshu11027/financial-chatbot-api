package mongodb

import (
	"context"
	"finance-chatbot/api/models"
	"fmt"
)


func CreateMessage(ctx context.Context, message *models.Message) error {
	collection := MongoClient.Database(MongoDatabase).Collection(MessageCollection)
	_, err := collection.InsertOne(ctx, message)
	if err != nil {
		return fmt.Errorf("error creating mongo item: %v", err)
	}
	return nil
}

func DeleteMessages(ctx context.Context, conversationID string) error {
	collection := MongoClient.Database(MongoDatabase).Collection(MessageCollection)
	_, err := collection.DeleteMany(ctx, map[string]interface{}{"conversation_id": conversationID})
	if err != nil {
		return fmt.Errorf("error deleting messages: %v", err)
	}
	return nil
}
