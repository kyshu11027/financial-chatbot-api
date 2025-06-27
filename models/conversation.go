package models

import (
	"time"

	"github.com/google/uuid"
)

type Context struct {
	ConversationID     string    `json:"conversation_id" bson:"conversation_id"`
	UserID             string    `json:"user_id" bson:"user_id"`
	Name               string    `json:"name" bson:"name"`
	CreatedAt          int64     `json:"created_at" bson:"created_at"`
	Income             float64   `json:"income" bson:"income"`
	SavingsGoal        float64   `json:"savings_goal" bson:"savings_goal"`
	AdditionalExpenses []Expense `json:"additional_monthly_expenses" bson:"additional_monthly_expenses"`
	Accounts           []Account `json:"accounts" bson:"accounts"`
}

type Message struct {
	ConversationID string `json:"conversation_id" bson:"conversation_id"`
	UserID         string `json:"user_id" bson:"user_id"`
	Text           string `json:"message" bson:"message"`
	Sender         string `json:"sender" bson:"sender"`
	Error          bool   `json:"error" bson:"error"`
	Timestamp      int64  `json:"timestamp" bson:"timestamp"`
}

type AIResponse struct {
	Message
	LastMessage bool `json:"last_message" bson:"last_message"`
}

type Conversation struct {
	ID        uuid.UUID `json:"id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	Title     string    `json:"title"`
}
