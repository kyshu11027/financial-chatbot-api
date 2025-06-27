package models

type User struct {
	UserID       string     `bson:"user_id" json:"user_id"`
	StripeID     *string    `bson:"stripe_id" json:"stripe_id"`
	Email        string     `bson:"email" json:"email"`
	Status       UserStatus `bson:"status" json:"status"`
	HasUsedTrial bool       `bson:"has_used_trial" json:"has_used_trial"`
}

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusTrial    UserStatus = "trial"
	UserStatusDeleted  UserStatus = "deleted"
)
