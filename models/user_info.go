package models

import "time"

type UserInfo struct {
	ID          string    `bson:"_id"`
	UserID      string    `bson:"user_id"`
	Income      float64   `bson:"income"`
	SavingsGoal float64   `bson:"savings_goal"`
	CreatedAt   time.Time `bson:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at"`
}
