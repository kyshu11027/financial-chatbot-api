package db

import (
	"database/sql"
	"finance-chatbot/api/models"
	"fmt"
)

// CreatePlaidItem creates a new Plaid item in the database
func CreatePlaidItem(userID, accessToken, itemID string) (*models.PlaidItem, error) {
	query := `
		INSERT INTO plaid_items (user_id, access_token, item_id, status)
		VALUES ($1, $2, $3, 'HEALTHY')
		RETURNING id, user_id, access_token, item_id, status, created_at, updated_at
	`

	item := &models.PlaidItem{}
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
func GetPlaidItemsByUserID(userID string) ([]*models.PlaidItem, error) {
	query := `
		SELECT id, user_id, access_token, item_id, status, created_at, updated_at, last_synced_at, sync_status, transaction_cursor
		FROM plaid_items
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := DB.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("error getting Plaid items: %v", err)
	}
	defer rows.Close()

	var items []*models.PlaidItem
	for rows.Next() {
		item := &models.PlaidItem{}
		err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.AccessToken,
			&item.ItemID,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.LastSyncedAt,
			&item.SyncStatus,
			&item.Cursor,
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
func GetPlaidItemByItemID(itemID string) (*models.PlaidItem, error) {
	query := `
		SELECT id, user_id, access_token, item_id, status, created_at, updated_at, last_synced_at, sync_status, transaction_cursor
		FROM plaid_items
		WHERE item_id = $1
	`

	item := &models.PlaidItem{}
	err := DB.QueryRow(query, itemID).Scan(
		&item.ID,
		&item.UserID,
		&item.AccessToken,
		&item.ItemID,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.LastSyncedAt,
		&item.SyncStatus,
		&item.Cursor,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting Plaid item: %v", err)
	}

	return item, nil
}

func UpdateSyncStatus(itemID string, syncStatus models.SyncStatus) error {
	query := `
		UPDATE plaid_items
        SET sync_status = $1, updated_at = CURRENT_TIMESTAMP
        WHERE item_id = $2;
	`
	result, err := DB.Exec(query, syncStatus, itemID)

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

func UpdateItemStatus(itemID string, status models.ItemStatus) error {
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
