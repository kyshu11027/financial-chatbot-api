package models

import (
	"database/sql"
	"fmt"
)

type Transaction struct {
	TransactionID string  `json:"transaction_id"`
	Date          string  `json:"date"`
	Amount        float64 `json:"amount"`
	Name          string  `json:"name"`
	MerchantName  string  `json:"merchant_name"`
	Category      string  `json:"category"`
	Pending       bool    `json:"pending"`
}

type Account struct {
	AccountID    string   `json:"account_id" bson:"account_id"`
	Name         string   `json:"name" bson:"name"`
	OfficialName string   `json:"official_name" bson:"official_name"`
	Type         string   `json:"type" bson:"type"`
	Subtype      string   `json:"subtype" bson:"subtype"`
	Mask         string   `json:"mask" bson:"mask"`
	Balances     Balances `json:"balances" bson:"balances"`
}

type Balances struct {
	Available       *float64 `json:"available" bson:"available"`
	Current         float64  `json:"current" bson:"current"`
	IsoCurrencyCode string   `json:"iso_currency_code" bson:"iso_currency_code"`
	Limit           *float64 `json:"limit" bson:"limit"`
}

type PlaidItem struct {
	ID           string       `json:"id"`
	UserID       string       `json:"user_id"`
	AccessToken  string       `json:"access_token"`
	ItemID       string       `json:"item_id"`
	Status       string       `json:"status"`
	CreatedAt    sql.NullTime `json:"created_at"`
	UpdatedAt    sql.NullTime `json:"updated_at"`
	LastSyncedAt sql.NullTime `json:"last_synced_at"`
	SyncStatus   SyncStatus   `json:"sync_status"`
	Cursor       *string      `json:"cursor"`
}

type PlaidError struct {
	ErrorType    string `json:"error_type"`
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	RequestId    string `json:"request_id"`
}

func (e *PlaidError) Error() string {
	return fmt.Sprintf("Plaid API error: %s (type: %s, code: %s, request_id: %s)",
		e.ErrorMessage, e.ErrorType, e.ErrorCode, e.RequestId)
}

type TransactionsJob struct {
	UserID      string  `json:"user_id"`
	AccessToken string  `json:"access_token"`
	ItemID      string  `json:"item_id"`
	Cursor      *string `json:"cursor"`
}

type SyncStatus string

const (
	TransactionsJobPending    SyncStatus = "pending"
	TransactionsJobFailed     SyncStatus = "failed"
	TransactionsJobInProgress SyncStatus = "in_progress"
	TransactionsJobIdle       SyncStatus = "idle"
)
