package models

import "time"

type UserInfo struct {
	ID          string    `bson:"_id" json:"id"`
	UserID      string    `bson:"user_id" json:"user_id"`
	Name        string    `bson:"name" json:"name"`
	Income      float64   `bson:"income" json:"income"`
	SavingsGoal float64   `bson:"savings_goal" json:"savings_goal"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at" json:"updated_at"`
}
