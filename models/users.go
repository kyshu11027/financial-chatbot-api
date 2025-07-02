package models

import "time"

type User struct {
	UserID             string     `bson:"user_id" json:"user_id"`
	StripeID           *string    `bson:"stripe_id" json:"stripe_id"`
	Email              string     `bson:"email" json:"email"`
	Status             UserStatus `bson:"status" json:"status"`
	SubscriptionID     *string    `bson:"subscription_id" json:"subscription_id"`
	HasUsedTrial       bool       `bson:"has_used_trial" json:"has_used_trial"`
	PlaidUserToken     *string    `bson:"plaid_user_token" json:"plaid_user_token"`
	ConsentRetrieved   bool       `bson:"consent_retrieved" json:"consent_retrieved"`
	ConsentRetrievedAt *time.Time  `bson:"consent_retrieved_at" json:"consent_retrieved_at"`
}

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusTrial    UserStatus = "trial"
	UserStatusDeleted  UserStatus = "deleted"
)
