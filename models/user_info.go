package models

type UserInfo struct {
	UserID             string    `bson:"user_id" json:"user_id"`
	Name               string    `bson:"name" json:"name"`
	Income             float64   `bson:"income" json:"income"`
	SavingsGoal        float64   `bson:"savings_goal" json:"savings_goal"`
	AdditionalExpenses []Expense `bson:"additional_monthly_expenses" json:"additional_monthly_expenses"`
	CreatedAt          int64     `bson:"created_at" json:"created_at"`
}

type Expense struct {
	Name        string `bson:"name" json:"name"`
	Amount      int    `bson:"amount" json:"amount"`
	Description string `bson:"description" json:"description"`
}
