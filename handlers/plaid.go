package handlers

import (
	"finance-chatbot/api/db"
	"finance-chatbot/api/logger"
	"finance-chatbot/api/models"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/plaid/plaid-go/v20/plaid"
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

type ProvisionTransactionsJobRequest struct {
	Items []models.PlaidItem `json:"items" binding:"required"`
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
	linkTokenRequest.SetWebhook(os.Getenv("PLAID_WEBHOOK_URL"))

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
		itemId := exchangeResponse.GetItemId()
		accessToken := exchangeResponse.GetAccessToken()
		_, err = db.CreatePlaidItem(
			claims.Sub,
			accessToken,
			itemId,
		)

		if err != nil {
			logger.Get().Error("error creating new plaid item",
				zap.String("user_id", claims.Sub),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Query transactions here and store in Qdrant as well as run a transactions/sync and store the cursor
		err = provisionSaveTransactionsJob(claims.Sub, itemId, accessToken, nil)

		if err != nil {
			logger.Get().Error("error provisioning transactions job",
				zap.String("access_token", accessToken),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	transactions, err := getTransactions(c, items)
	if err != nil {
		if plaidErr, ok := err.(*plaid.GenericOpenAPIError); ok {
			body := plaidErr.Body()
			logger.Get().Error("plaid API error raw body", zap.String("body", string(body)))
		} else {
			logger.Get().Error("error getting transactions",
				zap.String("user_id", claims.Sub),
				zap.Error(err))
		}
		return
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

func ProvisionTransactionsJob(c *gin.Context) {
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

	var req ProvisionTransactionsJobRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Get().Error("error binding JSON", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	items := req.Items

	for _, item := range items {
		if needsSync(item.LastSyncedAt, item.SyncStatus) {
			err := provisionSaveTransactionsJob(claims.Sub, item.ItemID, item.AccessToken, item.Cursor)

			if err != nil {
				logger.Get().Error("failed to produce transactions job request",
					zap.String("access_token", item.AccessToken),
					zap.Error(err))

				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func HandlePlaidWebhook(c *gin.Context) {
	logger.Get().Info("Received Plaid webhook")

	var webhook models.GenericPlaidWebhook
	if err := c.ShouldBindJSON(&webhook); err != nil {
		logger.Get().Error("error parsing generic webhook", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook payload"})
		return
	}

	logger.Get().Info("Parsed Plaid webhook",
		zap.String("webhook_type", webhook.WebhookType),
		zap.String("webhook_code", webhook.WebhookCode),
		zap.String("webhook_item_id", webhook.ItemID),
	)

	switch webhook.WebhookType {
	case "ITEM":
		switch webhook.WebhookCode {
		case "ERROR":
			if err := db.UpdateItemStatus(webhook.ItemID, "ERROR"); err != nil {
				logger.Get().Error("failed to update item status to ERROR", zap.Error(err))
			} else {
				logger.Get().Info("Updated item status to ERROR", zap.String("item_id", webhook.ItemID))
			}

		case "LOGIN_REPAIRED":
			logger.Get().Info("Item login repaired", zap.String("item_id", webhook.ItemID))

			if err := db.UpdateItemStatus(webhook.ItemID, "ACTIVE"); err != nil {
				logger.Get().Error("failed to update item status to ACTIVE", zap.Error(err))
			} else {
				logger.Get().Info("Updated item status to ACTIVE", zap.String("item_id", webhook.ItemID))
			}
		default:
			logger.Get().Info("Unhandled ITEM webhook code", zap.String("webhook_code", webhook.WebhookCode))
		}
	default:
		logger.Get().Info("Unhandled webhook type", zap.String("webhook_type", webhook.WebhookType))
	}

	logger.Get().Info("Plaid webhook processed successfully")
	c.JSON(http.StatusOK, gin.H{"status": "received"})
}
