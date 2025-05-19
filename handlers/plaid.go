package handlers

import (
	"bytes"
	"finance-chatbot/api/db"
	"finance-chatbot/api/logger"
	"finance-chatbot/api/models"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/plaid/plaid-go/plaid"
	"go.uber.org/zap"
)

var (
	PlaidClient *plaid.APIClient
)

type CreateLinkTokenRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

type ExchangeTokenRequest struct {
	PublicToken string `json:"public_token" binding:"required"`
}

type GetTransactionsRequest struct {
	AccessToken string `json:"access_token" binding:"required"`
}

func CreateLinkToken(c *gin.Context) {
	var req CreateLinkTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Get().Error("error binding JSON", zap.Error(err))
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

	logger.Get().Info("creating link token",
		zap.String("user_id", req.UserID),
		zap.Any("request", linkTokenRequest))

	linkToken, _, err := PlaidClient.PlaidApi.LinkTokenCreate(c.Request.Context()).LinkTokenCreateRequest(*linkTokenRequest).Execute()
	if err != nil {
		if plaidErr, ok := err.(*plaid.GenericOpenAPIError); ok {
			logger.Get().Error("plaid error",
				zap.String("error_body", string(plaidErr.Body())),
				zap.Error(plaidErr))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": plaidErr.Error(),
				"body":  string(plaidErr.Body()),
			})
		} else {
			logger.Get().Error("error creating link token", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	logger.Get().Info("link token created successfully",
		zap.String("user_id", req.UserID))
	c.JSON(http.StatusOK, gin.H{"link_token": linkToken.GetLinkToken()})
}

func ExchangePublicToken(c *gin.Context) {
	var req ExchangeTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Get().Error("error binding JSON", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		logger.Get().Error("user not authenticated")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	claims, ok := user.(*models.SupabaseClaims)
	if !ok {
		logger.Get().Error("invalid user claims")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user claims"})
		return
	}

	exchangeRequest := plaid.NewItemPublicTokenExchangeRequest(req.PublicToken)
	exchangeResponse, _, err := PlaidClient.PlaidApi.ItemPublicTokenExchange(c.Request.Context()).ItemPublicTokenExchangeRequest(*exchangeRequest).Execute()
	if err != nil {
		if plaidErr, ok := err.(*plaid.GenericOpenAPIError); ok {
			logger.Get().Error("plaid error",
				zap.String("error_body", string(plaidErr.Body())),
				zap.Error(plaidErr))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": plaidErr.Error(),
				"body":  string(plaidErr.Body()),
			})
		} else {
			logger.Get().Error("error exchanging public token", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	existingItem, err := db.GetPlaidItemByItemID(exchangeResponse.GetItemId())
	if err != nil {
		logger.Get().Error("error checking existing item",
			zap.String("item_id", exchangeResponse.GetItemId()),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if existingItem != nil {
		err = db.UpdatePlaidItemStatus(existingItem.ItemID, "active")
		if err != nil {
			logger.Get().Error("error updating existing item",
				zap.String("item_id", existingItem.ItemID),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update existing item"})
			return
		}
		logger.Get().Info("updated existing plaid item",
			zap.String("item_id", existingItem.ItemID),
			zap.String("user_id", claims.Sub))
	} else {
		_, err = db.CreatePlaidItem(
			claims.Sub,
			exchangeResponse.GetAccessToken(),
			exchangeResponse.GetItemId(),
		)
		if err != nil {
			logger.Get().Error("error creating new plaid item",
				zap.String("user_id", claims.Sub),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		logger.Get().Info("created new plaid item",
			zap.String("item_id", exchangeResponse.GetItemId()),
			zap.String("user_id", claims.Sub))
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": exchangeResponse.GetAccessToken(),
		"item_id":      exchangeResponse.GetItemId(),
	})
}

func GetTransactions(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Get().Error("failed to read request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	logger.Get().Debug("raw request body", zap.String("body", string(bodyBytes)))

	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var req GetTransactionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Get().Error("error binding JSON", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	endDate := time.Now().Format("2006-01-02")
	startDate := time.Now().AddDate(0, 0, -200).Format("2006-01-02")
	logger.Get().Info("fetching transactions",
		zap.String("start_date", startDate),
		zap.String("end_date", endDate))

	request := plaid.NewTransactionsGetRequest(
		req.AccessToken,
		startDate,
		endDate,
	)

	result, httpResp, err := PlaidClient.PlaidApi.TransactionsGet(c.Request.Context()).TransactionsGetRequest(*request).Execute()
	if err != nil {
		body, _ := io.ReadAll(httpResp.Body)
		logger.Get().Error("plaid API error",
			zap.String("response_body", string(body)),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	plaidTransactions := result.GetTransactions()
	logger.Get().Info("fetched transactions",
		zap.Int("count", len(plaidTransactions)))

	transactions := make([]models.Transaction, 0)
	for _, t := range plaidTransactions {
		transaction := models.Transaction{
			TransactionID: t.GetTransactionId(),
			Date:          t.GetDate(),
			Amount:        t.GetAmount(),
			Name:          t.GetName(),
			MerchantName:  t.GetMerchantName(),
			Category:      t.GetCategory(),
			Pending:       t.GetPending(),
		}
		transactions = append(transactions, transaction)
	}

	c.JSON(http.StatusOK, gin.H{"transactions": transactions})
}

func GetItems(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		logger.Get().Error("user not authenticated")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	claims, ok := user.(*models.SupabaseClaims)
	if !ok {
		logger.Get().Error("invalid user claims")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user claims"})
		return
	}

	items, err := db.GetPlaidItemsByUserID(claims.Sub)
	if err != nil {
		logger.Get().Error("error fetching plaid items",
			zap.String("user_id", claims.Sub),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Get().Info("fetched plaid items",
		zap.String("user_id", claims.Sub),
		zap.Int("item_count", len(items)))
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func GetItemsWithAccounts(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		logger.Get().Error("user not authenticated")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	claims, ok := user.(*models.SupabaseClaims)
	if !ok {
		logger.Get().Error("invalid user claims")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user claims"})
		return
	}

	items, err := db.GetPlaidItemsByUserID(claims.Sub)
	if err != nil {
		logger.Get().Error("error fetching plaid items",
			zap.String("user_id", claims.Sub),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type ItemWithAccounts struct {
		ItemID   string              `json:"item_id"`
		Accounts []plaid.AccountBase `json:"accounts"`
	}

	var response []ItemWithAccounts

	for _, item := range items {
		req := plaid.NewAccountsGetRequest(item.AccessToken)
		resp, _, err := PlaidClient.PlaidApi.AccountsGet(c.Request.Context()).AccountsGetRequest(*req).Execute()
		if err != nil {
			logger.Get().Error("failed to get accounts",
				zap.String("item_id", item.ItemID),
				zap.Error(err))
			continue
		}
		response = append(response, ItemWithAccounts{
			ItemID:   item.ItemID,
			Accounts: resp.GetAccounts(),
		})
	}

	c.JSON(http.StatusOK, gin.H{"items": response})
}
