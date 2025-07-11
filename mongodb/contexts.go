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

func UpdateConversationContext(ctx context.Context, conversationID string, updates map[string]any) error {
	collection := MongoClient.Database(MongoDatabase).Collection(ContextCollection)

	_, err := collection.UpdateOne(
		ctx,
		map[string]any{"conversation_id": conversationID},
		map[string]any{
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

func DeleteContextsByUserID(ctx context.Context, userID string) error {
	collection := MongoClient.Database(MongoDatabase).Collection(ContextCollection)

	filter := map[string]any{"user_id": userID}
	_, err := collection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("error deleting conversations for user_id %s: %v", userID, err)
	}

	return nil
}
