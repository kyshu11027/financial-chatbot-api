package db

import (
	"database/sql"
	"finance-chatbot/api/models"
	"fmt"
)

func UpdateStatusByUserID(userID, status models.UserStatus) error {
	query := `
		UPDATE users
		SET status = $1
		WHERE id = $2
	`
	_, err := DB.Exec(query, status, userID)
	if err != nil {
		return fmt.Errorf("error updating status for user %s: %v", userID, err)
	}
	return nil
}

func UpdateStripeIDByUserID(userID, stripeID string) error {
	query := `
		UPDATE users
		SET stripe_id = $1
		WHERE id = $2
	`
	_, err := DB.Exec(query, stripeID, userID)
	if err != nil {
		return fmt.Errorf("error updating Stripe ID for user %s: %v", userID, err)
	}
	return nil
}

func UpdateTrialStatusByStripeID(stripeID string, hasUsedTrial bool) error {
	query := `
		UPDATE users
		SET has_used_trial = $1
		WHERE stripe_id = $2
	`
	_, err := DB.Exec(query, hasUsedTrial, stripeID)
	if err != nil {
		return fmt.Errorf("error updating trial status for user %s: %v", stripeID, err)
	}
	return nil
}

func UpdateStatusByStripeID(stripeID string, status models.UserStatus) error {
	query := `
		UPDATE users
		SET status = $1
		WHERE stripe_id = $2
	`
	_, err := DB.Exec(query, status, stripeID)
	if err != nil {
		return fmt.Errorf("error updating status for Stripe ID %s: %v", stripeID, err)
	}
	return nil
}

func GetUserByID(userID string) (*models.User, error) {
	query := `
		SELECT id, stripe_id, status, email, has_used_trial
		FROM users
		WHERE id = $1
	`
	row := DB.QueryRow(query, userID)
	user := &models.User{}
	err := row.Scan(&user.UserID, &user.StripeID, &user.Status, &user.Email, &user.HasUsedTrial)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", userID)
		}
		return nil, fmt.Errorf("error getting user by ID %s: %v", userID, err)
	}
	return user, nil
}
