package handlers

import (
	"finance-chatbot/api/db"
	"finance-chatbot/api/logger"
	"finance-chatbot/api/models"
	"finance-chatbot/api/mongodb"
	"finance-chatbot/api/qdrant"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v82"
	"go.uber.org/zap"
)

func HandleDeleteUser(c *gin.Context) {
	logger.Get().Info("HandleDeleteUser called")

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

	_, err := unsubscribeFromStripe(claims.Sub)
	if err != nil {
		if stripeErr, ok := err.(*stripe.Error); ok {
			switch stripeErr.Code {
			case stripe.ErrorCodeResourceMissing:
				logger.Get().Warn("Stripe subscription not found (likely already deleted)", zap.String("user_id", claims.Sub))
			default:
				logger.Get().Error("Stripe error during unsubscribe", zap.String("code", string(stripeErr.Code)), zap.Error(err))
				c.JSON(http.StatusBadGateway, gin.H{"error": "Stripe error: " + stripeErr.Msg})
				return
			}
		} else {
			logger.Get().Error("Error unsubscribing from Stripe", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	accessTokens, err := db.DeleteUserDataByID(claims.Sub)
	if err != nil {
		logger.Get().Error("Error deleting user data stored in Postgres", zap.Error(err), zap.String("user_id", claims.Sub))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting user data stored in Postgres"})
	} else {
		logger.Get().Info("Deleted user data from Postgres", zap.String("user_id", claims.Sub))
	}

	err = DeletePlaidItems(c, accessTokens)
	if err != nil {
		logger.Get().Error("Error deleting plaid items from plaid", zap.Error(err), zap.String("user_id", claims.Sub))
		c.JSON(http.StatusInternalServerError, gin.H{"error removing plaid items from plaid": err.Error()})
	} else {
		logger.Get().Info("Deleted plaid items from plaid", zap.String("user_id", claims.Sub))
	}

	err = DeletePlaidUser(c)
	if err != nil {
		logger.Get().Error("Error deleting plaid user from plaid", zap.Error(err), zap.String("user_id", claims.Sub))
		c.JSON(http.StatusInternalServerError, gin.H{"error removing plaid user from plaid": err.Error()})
	} else {
		logger.Get().Info("Deleted plaid user from plaid", zap.String("user_id", claims.Sub))
	}

	err = mongodb.DeleteContextsByUserID(c.Request.Context(), claims.Sub)
	if err != nil {
		logger.Get().Error("Error deleting user conversation contexts", zap.Error(err), zap.String("user_id", claims.Sub))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting user conversation contexts"})
	} else {
		logger.Get().Info("Deleted user conversation contexts from MongoDB", zap.String("user_id", claims.Sub))
	}

	err = mongodb.DeleteUserInfo(c.Request.Context(), claims.Sub)
	if err != nil {
		logger.Get().Error("Error deleting user info", zap.Error(err), zap.String("user_id", claims.Sub))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting user info"})
	} else {
		logger.Get().Info("Deleted user info from MongoDB", zap.String("user_id", claims.Sub))
	}

	err = mongodb.DeleteMessagesByUserID(c.Request.Context(), claims.Sub)
	if err != nil {
		logger.Get().Error("Error deleting conversation messages", zap.Error(err), zap.String("user_id", claims.Sub))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting conversation messages"})
	} else {
		logger.Get().Info("Deleted conversation messages from MongoDB", zap.String("user_id", claims.Sub))
	}

	err = qdrant.DeleteTransactionsByUserID(claims.Sub)
	if err != nil {
		logger.Get().Error("Error deleting transactions from Qdrant", zap.Error(err), zap.String("user_id", claims.Sub))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting transactions from Qdrant"})
	} else {
		logger.Get().Info("Deleted transactions from Qdrant", zap.String("user_id", claims.Sub))
	}

	err = db.UpdateStatusToDeleteStateByUserID(claims.Sub)
	if err != nil {
		logger.Get().Error("Error updating user status", zap.Error(err), zap.String("user_id", claims.Sub))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating user status"})
	} else {
		logger.Get().Info("Updated user status to deleted and removed plaid user token", zap.String("user_id", claims.Sub))
	}

	err = DeleteSupabaseUser(claims.Sub)
	if err != nil {
		logger.Get().Error("Error deleting user from Supabase", zap.Error(err), zap.String("user_id", claims.Sub))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting user from Supabase"})
	} else {
		logger.Get().Info("Deleted user from Supabase", zap.String("user_id", claims.Sub))
	}

	logger.Get().Info("HandleDeleteUser completed successfully", zap.String("user_id", claims.Sub))
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func DeleteSupabaseUser(userID string) error {
	url := fmt.Sprintf("%s/auth/v1/admin/users/%s", os.Getenv("SUPABASE_URL"), userID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	serviceRoleKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	// Add authorization headers
	req.Header.Set("apikey", serviceRoleKey)
	req.Header.Set("Authorization", "Bearer "+serviceRoleKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code deleting user: %d", resp.StatusCode)
	}

	return nil
}
