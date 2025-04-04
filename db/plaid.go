package db

import (
	"database/sql"
	"fmt"
)

// PlaidItem represents a Plaid item in the database
type PlaidItem struct {
	ID            string
	UserID        string
	AccessToken   string
	ItemID        string
	Status        string
	CreatedAt     sql.NullTime
	UpdatedAt     sql.NullTime
}

// CreatePlaidItem creates a new Plaid item in the database
func CreatePlaidItem(userID, accessToken, itemID string) (*PlaidItem, error) {
	query := `
		INSERT INTO plaid_items (user_id, access_token, item_id, status)
		VALUES ($1, $2, $3, 'active')
		RETURNING id, user_id, access_token, item_id, status, created_at, updated_at
	`

	item := &PlaidItem{}
	err := DB.QueryRow(query, userID, accessToken, itemID).Scan(
		&item.ID,
		&item.UserID,
		&item.AccessToken,
		&item.ItemID,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating Plaid item: %v", err)
	}

	return item, nil
}

// GetPlaidItemsByUserID retrieves all Plaid items for a user
func GetPlaidItemsByUserID(userID string) ([]*PlaidItem, error) {
	query := `
		SELECT id, user_id, access_token, item_id, status, created_at, updated_at
		FROM plaid_items
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := DB.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("error getting Plaid items: %v", err)
	}
	defer rows.Close()

	var items []*PlaidItem
	for rows.Next() {
		item := &PlaidItem{}
		err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.AccessToken,
			&item.ItemID,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning Plaid item: %v", err)
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating Plaid items: %v", err)
	}

	return items, nil
}

// UpdatePlaidItemStatus updates the status of a Plaid item
func UpdatePlaidItemStatus(itemID, status string) error {
	query := `
		UPDATE plaid_items
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE item_id = $2
	`

	result, err := DB.Exec(query, status, itemID)
	if err != nil {
		return fmt.Errorf("error updating Plaid item status: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no Plaid item found with ID: %s", itemID)
	}

	return nil
}

// GetPlaidItemByItemID retrieves a Plaid item by its item_id
func GetPlaidItemByItemID(itemID string) (*PlaidItem, error) {
	query := `
		SELECT id, user_id, access_token, item_id, status, created_at, updated_at
		FROM plaid_items
		WHERE item_id = $1
	`

	item := &PlaidItem{}
	err := DB.QueryRow(query, itemID).Scan(
		&item.ID,
		&item.UserID,
		&item.AccessToken,
		&item.ItemID,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting Plaid item: %v", err)
	}

	return item, nil
} 