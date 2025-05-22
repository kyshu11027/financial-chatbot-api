package db

import "finance-chatbot/api/models"

func CreateConversation(userID string, title string) (*models.Conversation, error) {
	query := `
		INSERT INTO conversations (user_id, title)
		VALUES ($1, $2)
		RETURNING id, title, user_id, created_at
	`
	item := &models.Conversation{}

	err := DB.QueryRow(query, userID, title).Scan(
		&item.ID,
		&item.Title,
		&item.UserID,
		&item.CreatedAt,
	)
	if err != nil {
		return &models.Conversation{}, err
	}

	return item, nil
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

func GetByID(id string) (*models.Conversation, error) {
	query := `
		SELECT id, user_id, created_at, title
		FROM conversations
		WHERE id = $1
	`
	item := &models.Conversation{}
	err := DB.QueryRow(query, id).Scan(
		&item.ID,
		&item.UserID,
		&item.CreatedAt,
		&item.Title,
	)
	if err != nil {
		return nil, err
	}

	return item, nil
}

func Delete(id string) error {
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

func GetAllByUserID(userID string) ([]*models.Conversation, error) {
	query := `
		SELECT id, user_id, created_at, title
		FROM conversations
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	items := []*models.Conversation{}

	rows, err := DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		item := &models.Conversation{}
		err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.CreatedAt,
			&item.Title,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

func Update(id string, title string) (*models.Conversation, error) {
	query := `
		UPDATE conversations
		SET title = $1
		WHERE id = $2
		RETURNING id, user_id, created_at, title
	`

	item := &models.Conversation{}
	err := DB.QueryRow(query, title, id).Scan(
		&item.ID,
		&item.UserID,
		&item.CreatedAt,
		&item.Title,
	)
	if err != nil {
		return nil, err
	}

	return item, nil
}
