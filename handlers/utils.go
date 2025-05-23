package handlers

import (
	"context"
	"encoding/json"
	"finance-chatbot/api/db"
	"finance-chatbot/api/kafka"
	"finance-chatbot/api/logger"
	"finance-chatbot/api/models"
	"finance-chatbot/api/mongodb"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/plaid/plaid-go/plaid"
	"go.uber.org/zap"
)

func createConversationContext(c *gin.Context, userID string, conversationID string) (*models.Context, error) {
	logger.Get().Info("creating conversation context",
		zap.String("user_id", userID),
		zap.String("conversation_id", conversationID))

	items, err := db.GetPlaidItemsByUserID(userID)
	if err != nil {
		logger.Get().Error("error fetching plaid items",
			zap.String("user_id", userID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return nil, err
	}

	transactions, err := getTransactions(c, items)
	if err != nil {
		logger.Get().Error("error getting transactions",
			zap.String("user_id", userID),
			zap.Error(err))
		return nil, err
	}

	accounts, err := getAccounts(c, items)
	if err != nil {
		logger.Get().Error("error getting accounts",
			zap.String("user_id", userID),
			zap.Error(err))
		return nil, err
	}

	userInfo, err := getUserInfo(c, userID)
	if err != nil {
		logger.Get().Error("error getting user info",
			zap.String("user_id", userID),
			zap.Error(err))
		return nil, err
	}

	conversationContext := &models.Context{
		ConversationID:     conversationID,
		UserID:             userID,
		CreatedAt:          time.Now().Unix(),
		Transactions:       transactions,
		Accounts:           accounts,
		Income:             userInfo.Income,
		SavingsGoal:        userInfo.SavingsGoal,
		Name:               userInfo.Name,
		AdditionalExpenses: userInfo.AdditionalExpenses,
	}

	logger.Get().Info("conversation context created successfully",
		zap.String("user_id", userID),
		zap.String("conversation_id", conversationID))
	return conversationContext, nil
}

func getTransactions(c *gin.Context, items []*models.PlaidItem) ([]models.Transaction, error) {

	endDate := time.Now().Format("2006-01-02")
	startDate := time.Now().AddDate(0, 0, -180).Format("2006-01-02")
	transactions := []models.Transaction{}

	logger.Get().Info("fetching transactions",
		zap.String("start_date", startDate),
		zap.String("end_date", endDate))

	for _, plaidItem := range items {
		logger.Get().Debug("fetching transactions for plaid item",
			zap.String("access_token", plaidItem.AccessToken))

		request := plaid.NewTransactionsGetRequest(
			plaidItem.AccessToken,
			startDate,
			endDate,
		)

		result, _, err := PlaidClient.PlaidApi.TransactionsGet(c.Request.Context()).TransactionsGetRequest(*request).Execute()
		if err != nil {
			logger.Get().Error("error fetching transactions for plaid item",
				zap.String("access_token", plaidItem.AccessToken),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return nil, err
		}

		plaidTransactions := result.GetTransactions()
		logger.Get().Info("transactions fetched successfully",
			zap.String("access_token", plaidItem.AccessToken),
			zap.Int("transaction_count", len(plaidTransactions)))

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
	}

	logger.Get().Info("all transactions retrieved successfully",
		zap.Int("total_transactions", len(transactions)))
	return transactions, nil
}

func getAccounts(c *gin.Context, items []*models.PlaidItem) ([]models.Account, error) {
	var accounts []models.Account

	for _, item := range items {
		req := plaid.NewAccountsGetRequest(item.AccessToken)
		resp, _, err := PlaidClient.PlaidApi.
			AccountsGet(c.Request.Context()).
			AccountsGetRequest(*req).
			Execute()
		if err != nil {
			logger.Get().Error("failed to get accounts",
				zap.String("item_id", item.ItemID),
				zap.Error(err))
			continue
		}

		for _, acct := range resp.GetAccounts() {
			plaidBalances := acct.GetBalances()

			account := models.Account{
				AccountID:    acct.GetAccountId(),
				Name:         acct.GetName(),
				OfficialName: acct.GetOfficialName(),
				Type:         string(acct.GetType()),
				Subtype:      string(acct.GetSubtype()),
				Mask:         acct.GetMask(),
			}

			var available *float32
			if plaidBalances.Available.IsSet() {
				v := plaidBalances.Available.Get()
				available = v
			}

			var limit *float32
			if plaidBalances.Limit.IsSet() {
				v := plaidBalances.Limit.Get()
				limit = v
			}

			account.Balances = models.Balances{
				Available:       available,
				Current:         plaidBalances.GetCurrent(),
				IsoCurrencyCode: plaidBalances.GetIsoCurrencyCode(),
				Limit:           limit,
			}

			accounts = append(accounts, account)
		}
	}

	return accounts, nil
}

func getUserInfo(c *gin.Context, userID string) (*models.UserInfo, error) {
	userInfo, err := mongodb.GetUserInfo(c, userID)
	if err != nil {
		logger.Get().Error("error fetching user info",
			zap.String("user_id", userID),
			zap.Error(err))
		return nil, err
	}

	return userInfo, nil
}

func processUserMessage(ctx context.Context, userId string, msg *models.Message) error {
	msg.UserID = userId
	msg.Sender = "UserMessage"
	msg.Timestamp = time.Now().Unix()

	err := mongodb.CreateMessage(ctx, msg)
	if err != nil {
		logger.Get().Error("failed to create message",
			zap.String("user_id", userId),
			zap.Error(err))
		return fmt.Errorf("failed to create message: %w", err)
	}

	messageBytes, err := json.Marshal(msg)
	if err != nil {
		logger.Get().Error("failed to marshal message",
			zap.String("user_id", userId),
			zap.Error(err))
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = kafka.ProduceMessage(kafka.MessageTopic, messageBytes)
	if err != nil {
		logger.Get().Error("failed to produce message",
			zap.String("user_id", userId),
			zap.Error(err))
		return fmt.Errorf("failed to produce message: %w", err)
	}

	return nil
}

func authenticateSSE(c *gin.Context) error {
	tokenString := c.DefaultQuery("token", "")
	if tokenString == "" {
		logger.Get().Error("missing or invalid token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid token"})
		c.Abort()
		return fmt.Errorf("missing or invalid token")
	}

	claims := &models.SupabaseClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		secret := os.Getenv("SUPABASE_JWT_SECRET")
		if secret == "" {
			return nil, fmt.Errorf("SUPABASE_JWT_SECRET environment variable not set")
		}
		return []byte(secret), nil
	})

	if err != nil {
		logger.Get().Error("error parsing claims", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
		c.Abort()
		return err
	}

	if !token.Valid {
		logger.Get().Error("invalid token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		c.Abort()
		return err
	}

	if claims.Issuer != os.Getenv("SUPABASE_URL")+"/auth/v1" {
		logger.Get().Error("invalid token issuer",
			zap.String("issuer", claims.Issuer))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token issuer"})
		c.Abort()
		return err
	}
	return nil
}
