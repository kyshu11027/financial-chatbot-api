package models

type Transaction struct {
	TransactionID string   `json:"transaction_id"`
	Date          string   `json:"date"`
	Amount        float32  `json:"amount"`
	Name          string   `json:"name"`
	MerchantName  string   `json:"merchant_name"`
	Category      []string `json:"category"`
	Pending       bool     `json:"pending"`
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
	Available       *float32 `json:"available" bson:"available"`
	Current         float32  `json:"current" bson:"current"`
	IsoCurrencyCode string   `json:"iso_currency_code" bson:"iso_currency_code"`
	Limit           *float32 `json:"limit" bson:"limit"`
}
