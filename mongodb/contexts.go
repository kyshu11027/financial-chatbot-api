package mongodb

import (
	"context"
	"finance-chatbot/api/models"
	"fmt"
)

func CreateConversationContext(ctx context.Context, item *models.Context) error {
	collection := MongoClient.Database(MongoDatabase).Collection(ContextCollection)
	_, err := collection.InsertOne(ctx, item)
	if err != nil {
		return fmt.Errorf("error creating mongo item: %v", err)
	}
	return nil
}

func UpdateConversationContext(ctx context.Context, conversationID string, updates map[string]interface{}) error {
	collection := MongoClient.Database(MongoDatabase).Collection(ContextCollection)

	_, err := collection.UpdateOne(
		ctx,
		map[string]interface{}{"conversation_id": conversationID},
		map[string]interface{}{
			"$set": updates,
		},
	)
	if err != nil {
		return fmt.Errorf("error updating mongo item: %v", err)
	}
	return nil
}

func DeleteConversation(ctx context.Context, conversationID string) error {
	collection := MongoClient.Database(MongoDatabase).Collection(ContextCollection)

	_, err := collection.DeleteMany(ctx, map[string]any{"conversation_id": conversationID})
	if err != nil {
		return fmt.Errorf("error deleting mongo items: %v", err)
	}
	return nil
}
