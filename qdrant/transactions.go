package qdrant

import (
	"context"
	"fmt"

	"github.com/qdrant/go-client/qdrant"
)

// DeleteTransactionsByUserID deletes all transactions from the "transactions" collection
// that have metadata field "user_id" equal to the given userId.
func DeleteTransactionsByUserID(userId string) error {
	if QdrantClient == nil {
		return fmt.Errorf("QdrantClient is not initialized")
	}

	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "metadata.user_id",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: userId,
							},
						},
					},
				},
			},
		},
	}

	waitBeforeReturning := false
	_, err := QdrantClient.Delete(context.Background(), &qdrant.DeletePoints{
		CollectionName: TransactionsCollection,
		Points:         qdrant.NewPointsSelectorFilter(filter),
		Wait:           &waitBeforeReturning, 
	})
	if err != nil {
		return fmt.Errorf("failed to delete transactions for user_id %s: %w", userId, err)
	}

	return nil
}
