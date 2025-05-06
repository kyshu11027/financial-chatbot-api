package db

import (
	"context"
	"database/sql"
	"finance-chatbot/api/models"
)

func CreateUserInfo(ctx context.Context, item *models.UserInfo) error {
	query := `
		INSERT INTO user_info (user_id, name, income, savings_goal)
		VALUES ($1, $2, $3, $4)
		RETURNING user_id, name, income, savings_goal
	`

	err := DB.QueryRow(query, item.UserID, item.Name, item.Income, item.SavingsGoal).Scan(
		&item.UserID,
		&item.Name,
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
		SET name = $1, income = $2, savings_goal = $3
		WHERE user_id = $4
	`

	_, err := DB.Exec(query, updates.Name, updates.Income, updates.SavingsGoal, id)
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
		SELECT id, user_id, income, savings_goal, name, created_at, updated_at FROM user_info WHERE user_id = $1
	`
	item := &models.UserInfo{}
	err := DB.QueryRow(query, userID).Scan(
		&item.ID,
		&item.UserID,
		&item.Income,
		&item.SavingsGoal,
		&item.Name,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No rows found, return nil
		}
		return nil, err
	}

	return item, nil
}
