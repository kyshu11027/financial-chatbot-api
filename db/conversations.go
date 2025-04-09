package db

import (
	"time"

	"github.com/google/uuid"
)

type Conversation struct {
	ID        uuid.UUID `json:"id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

func CreateConversation(userID string) (string, error) {
	query := `
		INSERT INTO conversations (user_id)
		VALUES ($1)
		RETURNING id, user_id, created_at
	`
	item := &Conversation{}

	err := DB.QueryRow(query, userID).Scan(
		&item.ID,
		&item.UserID,
		&item.CreatedAt,
	)
	if err != nil {
		return "", err
	}

	return item.ID.String(), nil
}

func DeleteConversation(id string) error {
	query := `
		DELETE FROM conversations
		WHERE id = $1
	`
	_, err := DB.Exec(query, id)
	if err != nil {
		return err
	}

	return nil
}

func GetByID(id uuid.UUID) (*Conversation, error) {
	query := `
		SELECT id, user_id, created_at
		FROM conversations
		WHERE id = $1
	`
	item := &Conversation{}
	err := DB.QueryRow(query, id).Scan(
		&item.ID,
		&item.UserID,
		&item.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return item, nil
}

func Delete(id uuid.UUID) error {
	query := `
		DELETE FROM conversations
		WHERE id = $1
	`
	_, err := DB.Exec(query, id)
	if err != nil {
		return err
	}

	return nil
}

func GetAllByUserID(userID string) ([]*Conversation, error) {
	query := `
		SELECT id, user_id, created_at
		FROM conversations
		WHERE user_id = $1
	`
	items := []*Conversation{}

	rows, err := DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		item := &Conversation{}
		err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}
