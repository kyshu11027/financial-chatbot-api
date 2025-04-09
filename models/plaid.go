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
