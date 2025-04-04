package handlers

import (
	"finance-chatbot/api/db"
	"finance-chatbot/api/middleware"
	"net/http"
	"time"

	"log"

	"github.com/gin-gonic/gin"
	"github.com/plaid/plaid-go/plaid"
)

var PlaidClient *plaid.APIClient

type CreateLinkTokenRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

type ExchangeTokenRequest struct {
	PublicToken string `json:"public_token" binding:"required"`
}

func CreateLinkToken(c *gin.Context) {
	var req CreateLinkTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	linkTokenRequest := plaid.NewLinkTokenCreateRequest(
		"Finance Chatbot",
		"en",
		[]plaid.CountryCode{plaid.COUNTRYCODE_US},
		plaid.LinkTokenCreateRequestUser{
			ClientUserId: req.UserID,
		},
	)
	linkTokenRequest.SetProducts([]plaid.Products{plaid.PRODUCTS_TRANSACTIONS})

	// Log the request details
	log.Printf("Creating link token for user: %s", req.UserID)
	log.Printf("Request: %+v", linkTokenRequest)

	linkToken, _, err := PlaidClient.PlaidApi.LinkTokenCreate(c.Request.Context()).LinkTokenCreateRequest(*linkTokenRequest).Execute()
	if err != nil {
		if plaidErr, ok := err.(*plaid.GenericOpenAPIError); ok {
			log.Printf("Plaid error: %s", string(plaidErr.Body()))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": plaidErr.Error(),
				"body":  string(plaidErr.Body()),
			})
		} else {
			log.Printf("Error creating link token: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"link_token": linkToken.GetLinkToken()})
}

func ExchangePublicToken(c *gin.Context) {
	var req ExchangeTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (set by auth middleware)
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	claims, ok := user.(*middleware.SupabaseClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user claims"})
		return
	}

	exchangeRequest := plaid.NewItemPublicTokenExchangeRequest(req.PublicToken)
	exchangeResponse, _, err := PlaidClient.PlaidApi.ItemPublicTokenExchange(c.Request.Context()).ItemPublicTokenExchangeRequest(*exchangeRequest).Execute()
	if err != nil {
		if plaidErr, ok := err.(*plaid.GenericOpenAPIError); ok {
			log.Printf("Plaid error: %s", string(plaidErr.Body()))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": plaidErr.Error(),
				"body":  string(plaidErr.Body()),
			})
		} else {
			log.Printf("Error exchanging public token: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// Check if item already exists
	existingItem, err := db.GetPlaidItemByItemID(exchangeResponse.GetItemId())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if existingItem != nil {
		// Update existing item
		err = db.UpdatePlaidItemStatus(existingItem.ItemID, "active")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update existing item"})
			return
		}
	} else {
		// Create new item
		_, err = db.CreatePlaidItem(
			claims.Sub,
			exchangeResponse.GetAccessToken(),
			exchangeResponse.GetItemId(),
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": exchangeResponse.GetAccessToken(),
		"item_id":     exchangeResponse.GetItemId(),
	})
}

type GetTransactionsRequest struct {
	AccessToken string `json:"access_token" binding:"required"`
}

func GetTransactions(c *gin.Context) {
	var req GetTransactionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get transactions from the last 30 days
	endDate := time.Now().Format("2006-01-02")
	startDate := time.Now().AddDate(0, 0, -200).Format("2006-01-02")

	request := plaid.NewTransactionsGetRequest(
		req.AccessToken,
		startDate,
		endDate,
	)

	result, _, err := PlaidClient.PlaidApi.TransactionsGet(c.Request.Context()).TransactionsGetRequest(*request).Execute()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	transactions := result.GetTransactions()

	// Format transactions for response
	formattedTransactions := make([]map[string]interface{}, 0)
	for _, t := range transactions {
		transaction := map[string]interface{}{
			"transaction_id": t.GetTransactionId(),
			"date":          t.GetDate(),
			"amount":        t.GetAmount(),
			"name":          t.GetName(),
			"merchant_name": t.GetMerchantName(),
			"category":      t.GetCategory(),
			"pending":       t.GetPending(),
		}
		formattedTransactions = append(formattedTransactions, transaction)
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": formattedTransactions,
	})
}

func GetItems(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	claims, ok := user.(*middleware.SupabaseClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user claims"})
		return
	}

	items, err := db.GetPlaidItemsByUserID(claims.Sub)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

