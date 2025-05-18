package mongodb

import (
	"context"
	"finance-chatbot/api/models"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func CreateUserInfo(ctx context.Context, item *models.UserInfo) error {
	collection := MongoClient.Database(MongoDatabase).Collection(UserInfoCollection)
	_, err := collection.InsertOne(ctx, item)
	if err != nil {
		return fmt.Errorf("error creating mongo item: %v", err)
	}
	return nil
}

func ReplaceUserInfo(ctx context.Context, userID string, info *models.UserInfo) error {
	collection := MongoClient.Database(MongoDatabase).Collection(UserInfoCollection)

	filter := bson.M{"user_id": userID}
	_, err := collection.ReplaceOne(ctx, filter, info)
	if err != nil {
		return fmt.Errorf("error replacing user info: %w", err)
	}

	return nil
}

func GetUserInfo(ctx context.Context, userID string) (*models.UserInfo, error) {
	collection := MongoClient.Database(MongoDatabase).Collection(UserInfoCollection)
	filter := bson.M{
		"user_id": userID,
	}

	var userInfo models.UserInfo
	err := collection.FindOne(ctx, filter).Decode(&userInfo)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Not found, but not an error
		}
		return nil, err
	}

	return &userInfo, nil
}

func DeleteUserInfo(ctx context.Context, userID string) error {
	collection := MongoClient.Database(MongoDatabase).Collection(UserInfoCollection)

	_, err := collection.DeleteMany(ctx, map[string]any{"user_id": userID})
	if err != nil {
		return fmt.Errorf("error deleting mongo items: %v", err)
	}
	return nil
}
