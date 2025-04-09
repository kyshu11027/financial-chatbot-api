package handlers

import (
	"finance-chatbot/api/db"
	"finance-chatbot/api/middleware"
	"finance-chatbot/api/models"
	"finance-chatbot/api/mongodb"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/plaid/plaid-go/plaid"
)

func HandleCreateNewChat(c *gin.Context) {
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

	conversationID, err := db.CreateConversation(claims.Sub)
	if err != nil {
		log.Printf("Error creating conversation for user %s: %v", claims.Sub, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	conversationContext, err := createConversationContext(c, claims.Sub, conversationID)
	if err != nil {
		log.Printf("Error creating conversation context for user %s: %v", claims.Sub, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = mongodb.CreateConversationContext(c, conversationContext)
	if err != nil {
		log.Printf("Error saving conversation context to MongoDB for conversation ID %s: %v", conversationID, err)

		err = db.DeleteConversation(conversationID)
		if err != nil {
			log.Printf("Error deleting conversation from DB for conversation ID %s: %v", conversationID, err)
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Successfully created new chat for user %s with conversation ID %s", claims.Sub, conversationID)
	c.JSON(http.StatusOK, gin.H{"conversation_id": conversationID, "conversation_context": conversationContext})
}

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
