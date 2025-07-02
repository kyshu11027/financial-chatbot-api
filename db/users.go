package db

import (
	"database/sql"
	"finance-chatbot/api/models"
	"fmt"
)

func UpdateStatusByUserID(userID string, status models.UserStatus) error {
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

func UpdateStatusByStripeID(stripeID string, status models.UserStatus, subscriptionID *string) error {

	var err error
	if subscriptionID == nil {
		query := `
			UPDATE users
			SET status = $1
			WHERE stripe_id = $2
		`
		_, err = DB.Exec(query, status, stripeID)
	} else {
		query := `
			UPDATE users
			SET status = $1, subscription_id = $2
			WHERE stripe_id = $3
		`
		_, err = DB.Exec(query, status, *subscriptionID, stripeID)
	}

	if err != nil {
		return fmt.Errorf("error updating status for Stripe ID %s: %v", stripeID, err)
	}
	return nil
}

func UpdatePlaidUserTokenByUserID(userID string, plaidUserToken string) error {
	query := `
		UPDATE users
		SET plaid_user_token = $1
		WHERE id = $2
	`
	_, err := DB.Exec(query, plaidUserToken, userID)
	if err != nil {
		return fmt.Errorf("error updating trial status for user %s: %v", userID, err)
	}
	return nil
}

func GetUserByID(userID string) (*models.User, error) {
	query := `
		SELECT id, stripe_id, status, email, has_used_trial, subscription_id, plaid_user_token, consent_retrieved, consent_retrieved_at
		FROM users
		WHERE id = $1
	`
	row := DB.QueryRow(query, userID)
	user := &models.User{}
	err := row.Scan(&user.UserID, &user.StripeID, &user.Status, &user.Email, &user.HasUsedTrial, &user.SubscriptionID, &user.PlaidUserToken, &user.ConsentRetrieved, &user.ConsentRetrievedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", userID)
		}
		return nil, fmt.Errorf("error getting user by ID %s: %v", userID, err)
	}
	return user, nil
}

func DeleteUserDataByID(userID string) (accessTokens []string, err error) {
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// Delete plaid_items and return access_tokens
	rows, err := tx.Query(`DELETE FROM plaid_items WHERE user_id = $1 RETURNING access_token`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, err
		}
		accessTokens = append(accessTokens, token)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Delete conversations
	if _, err = tx.Exec(`DELETE FROM conversations WHERE user_id = $1`, userID); err != nil {
		return nil, err
	}

	return accessTokens, nil
}
