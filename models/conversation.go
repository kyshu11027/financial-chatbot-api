package models

type Context struct {
	ConversationID string        `json:"conversation_id" bson:"conversation_id"`
	UserID         string        `json:"user_id" bson:"user_id"`
	CreatedAt      int64         `json:"created_at" bson:"created_at"`
	Income         float64       `json:"income" bson:"income"`
	SavingsGoal    float64       `json:"savings_goal" bson:"savings_goal"`
	Transactions   []Transaction `json:"transactions" bson:"transactions"`
}

type Message struct {
	ConversationID string `json:"conversation_id" bson:"conversation_id"`
	UserID         string `json:"user_id" bson:"user_id"`
	Message        string `json:"message" bson:"message"`
	Sender         string `json:"sender" bson:"sender"`
}
