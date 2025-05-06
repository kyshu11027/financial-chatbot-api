package models

type Context struct {
	ConversationID string        `json:"conversation_id" bson:"conversation_id"`
	UserID         string        `json:"user_id" bson:"user_id"`
	Name           string        `json:"name" bson:"name"`
	CreatedAt      int64         `json:"created_at" bson:"created_at"`
	Income         float64       `json:"income" bson:"income"`
	SavingsGoal    float64       `json:"savings_goal" bson:"savings_goal"`
	Transactions   []Transaction `json:"transactions" bson:"transactions"`
}

type Message struct {
	ConversationID string `json:"conversation_id" bson:"conversation_id"`
	UserID         string `json:"user_id" bson:"user_id"`
	Text           string `json:"message" bson:"message"`
	Sender         string `json:"sender" bson:"sender"`
	Timestamp      int64  `json:"timestamp" bson:"timestamp"`
}

type AIResponse struct {
	Message
	LastMessage bool `json:"last_message" bson:"last_message"`
}

type NewChat struct {
	Message string `json:"message" bson:"message"`
}
