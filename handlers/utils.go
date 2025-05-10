package handlers

import (
	"context"
	"encoding/json"
	"finance-chatbot/api/db"
	"finance-chatbot/api/kafka"
	"finance-chatbot/api/models"
	"finance-chatbot/api/mongodb"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/plaid/plaid-go/plaid"
)

func createConversationContext(c *gin.Context, userID string, conversationID string) (*models.Context, error) {
	log.Printf("Creating conversation context for userID: %s, conversationID: %s", userID, conversationID)
	transactions, err := getTransactions(c, userID)
	if err != nil {
		log.Printf("Error getting transactions for userID %s: %v", userID, err)
		return nil, err
	}

	userInfo, err := getUserInfo(c, userID)
	if err != nil {
		log.Printf("Error getting user info for userID %s: %v", userID, err)
		return nil, err
	}

	conversationContext := &models.Context{
		ConversationID: conversationID,
		UserID:         userID,
		CreatedAt:      time.Now().Unix(),
		Transactions:   transactions,
		Income:         userInfo.Income,
		SavingsGoal:    userInfo.SavingsGoal,
		Name:           userInfo.Name,
	}
	log.Printf("Successfully created conversation context for userID: %s, conversationID: %s", userID, conversationID)

	return conversationContext, nil
}

func getTransactions(c *gin.Context, userID string) ([]models.Transaction, error) {
	log.Printf("Fetching plaid items for userID: %s", userID)
	plaidItems, err := db.GetPlaidItemsByUserID(userID)
	if err != nil {
		log.Printf("Error fetching plaid items for userID %s: %v", userID, err)
		return nil, err
	}

	// Get transactions from the last 180 days
	endDate := time.Now().Format("2006-01-02")
	startDate := time.Now().AddDate(0, 0, -180).Format("2006-01-02")
	transactions := []models.Transaction{}
	log.Printf("Fetching transactions from %s to %s for userID: %s", startDate, endDate, userID)

	for _, plaidItem := range plaidItems {
		log.Printf("Fetching transactions for plaid item with access token: %s", plaidItem.AccessToken)
		request := plaid.NewTransactionsGetRequest(
			plaidItem.AccessToken,
			startDate,
			endDate,
		)

		result, _, err := PlaidClient.PlaidApi.TransactionsGet(c.Request.Context()).TransactionsGetRequest(*request).Execute()
		if err != nil {
			log.Printf("Error fetching transactions for plaid item with access token %s: %v", plaidItem.AccessToken, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return nil, err
		}
		plaidTransactions := result.GetTransactions()
		log.Printf("Successfully fetched %d transactions for plaid item with access token: %s", len(plaidTransactions), plaidItem.AccessToken)

		// Format transactions for response
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

	log.Printf("Successfully retrieved %d transactions for userID: %s", len(transactions), userID)
	return transactions, nil
}

func getUserInfo(c *gin.Context, userID string) (*models.UserInfo, error) {
	userInfo, err := db.GetUserInfo(c, userID)
	if err != nil {
		log.Printf("Error fetching user info for userID %s: %v", userID, err)
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
		return fmt.Errorf("failed to create message: %w", err)
	}

	messageBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = kafka.ProduceMessage(kafka.MessageTopic, messageBytes)
	if err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	return nil
}

func authenticateSSE(c *gin.Context) error {
	tokenString := c.DefaultQuery("token", "")
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid token"})
		c.Abort()
		return fmt.Errorf("missing or invalid token")
	}

	claims := &models.SupabaseClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method is HS256
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		// Use the JWT secret for verification
		secret := os.Getenv("SUPABASE_JWT_SECRET")
		if secret == "" {
			return nil, fmt.Errorf("SUPABASE_JWT_SECRET environment variable not set")
		}
		return []byte(secret), nil
	})

	if err != nil {
		log.Printf("Error parsing claims: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
		c.Abort()
		return err
	}

	if !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		c.Abort()
		return err
	}

	// Verify issuer
	if claims.Issuer != os.Getenv("SUPABASE_URL")+"/auth/v1" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token issuer"})
		c.Abort()
		return err
	}
	return nil
}
