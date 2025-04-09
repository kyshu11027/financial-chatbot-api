package db

import (
	"context"
	"finance-chatbot/api/models"
)

func CreateUserInfo(ctx context.Context, item *models.UserInfo) error {
	query := `
		INSERT INTO user_info (user_id, income, savings_goal)
		VALUES ($1, $2, $3)
		RETURNING user_id, income, savings_goal
	`

	err := DB.QueryRow(query, item.UserID, item.Income, item.SavingsGoal).Scan(
		&item.UserID,
		&item.Income,
		&item.SavingsGoal,
	)

	if err != nil {
		return err
	}

	return nil
}

func UpdateUserInfo(ctx context.Context, id string, updates *models.UserInfo) error {
	query := `
		UPDATE user_info
		SET income = $1, savings_goal = $2
		WHERE id = $3
	`

	_, err := DB.Exec(query, updates.Income, updates.SavingsGoal, id)
	if err != nil {
		return err
	}

	return nil
}

func DeleteUserInfo(ctx context.Context, userID string) error {
	query := `
		DELETE FROM user_info WHERE user_id = $1
	`
	_, err := DB.Exec(query, userID)
	if err != nil {
		return err
	}

	return nil
}

func GetUserInfo(ctx context.Context, userID string) (*models.UserInfo, error) {
	query := `
		SELECT id, user_id, income, savings_goal, created_at, updated_at FROM user_info WHERE user_id = $1
	`
	item := &models.UserInfo{}
	err := DB.QueryRow(query, userID).Scan(
		&item.ID,
		&item.UserID,
		&item.Income,
		&item.SavingsGoal,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return item, nil
}
