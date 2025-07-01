package handlers

import (
	"encoding/json"
	"finance-chatbot/api/db"
	"finance-chatbot/api/logger"
	"finance-chatbot/api/middleware"
	"finance-chatbot/api/models"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/customer"
	"github.com/stripe/stripe-go/v82/subscription"
	"go.uber.org/zap"
)

func HandleCreateStripeSession(c *gin.Context) {
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

	userInfo, err := db.GetUserByID(claims.Sub)
	logger.Get().Debug("User info retrieved", zap.String("user_id", claims.Sub), zap.Any("user_info", userInfo))
	if err != nil {
		logger.Get().Error("Failed to get user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check email existence"})
		return
	}

	subscriptionData := &stripe.CheckoutSessionSubscriptionDataParams{}
	if !userInfo.HasUsedTrial {
		logger.Get().Debug("User has not used free trial, adding trial period")
		subscriptionData.TrialPeriodDays = stripe.Int64(1)
	}

	customerParams := &stripe.CustomerParams{
		Email: stripe.String(claims.Email),
	}

	cust, err := customer.New(customerParams)
	if err != nil {
		logger.Get().Error("Failed to create Stripe customer", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Stripe customer"})
		return
	}

	params := &stripe.CheckoutSessionParams{
		Customer:   stripe.String(cust.ID),
		SuccessURL: stripe.String(string(os.Getenv("CLIENT_URL"))),
		CancelURL:  stripe.String(string(os.Getenv("CLIENT_URL"))),
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(os.Getenv("STRIPE_PRICE_ID")),
				Quantity: stripe.Int64(1),
			},
		},
		SubscriptionData: subscriptionData,
	}

	s, err := session.New(params)

	if err != nil {
		logger.Get().Error("Failed to create Stripe session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Stripe session creation failed"})
		return
	}

	if err := db.UpdateStripeIDByUserID(claims.Sub, cust.ID); err != nil {
		logger.Get().Error("Failed to update Stripe ID in database", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update Stripe ID in database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url": s.URL,
	})
}

// func HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
func HandleStripeWebhook(c *gin.Context) {
	eventRaw, exists := c.Get(middleware.StripeEventKey)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Missing Stripe event in context"})
		return
	}

	event, ok := eventRaw.(stripe.Event)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid event type"})
		return
	}

	var stripeID string

	switch event.Type {
	case "checkout.session.completed":
		logger.Get().Info("Checkout session completed", zap.String("event_id", event.ID))
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			logger.Get().Error("Error parsing session", zap.Error(err))
			c.Status(http.StatusBadRequest)
			return
		}
		stripeID = session.Customer.ID
		logger.Get().Debug("User IDs", zap.String("stripe_id", stripeID))
		if err := db.UpdateTrialStatusByStripeID(stripeID, true); err != nil {
			logger.Get().Error("Error updating Stripe ID", zap.Error(err))
			c.Status(http.StatusInternalServerError)
			return
		}

	case "customer.subscription.created":
		logger.Get().Info("Customer subscription created", zap.String("event_type", string(event.Type)), zap.String("event_id", event.ID))
		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			logger.Get().Error("Error parsing subscription", zap.Error(err))
			c.Status(http.StatusBadRequest)
			return
		}
		stripeID = subscription.Customer.ID
		if err := db.UpdateStatusByStripeID(stripeID, models.UserStatusTrial, &subscription.ID); err != nil {
			logger.Get().Error("Error updating user status", zap.Error(err))
			c.Status(http.StatusInternalServerError)
			return
		}

	case "invoice.paid", "invoice.payment_failed":
		logger.Get().Info("Invoice event received", zap.String("event_type", string(event.Type)), zap.String("event_id", event.ID))
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			logger.Get().Error("Error parsing invoice", zap.Error(err))
			c.Status(http.StatusBadRequest)
			return
		}
		stripeID = invoice.Customer.ID

		status := models.UserStatusActive
		if event.Type == "invoice.payment_failed" {
			status = models.UserStatusInactive
		}

		if err := db.UpdateStatusByStripeID(stripeID, status, nil); err != nil {
			logger.Get().Error("Error updating user status", zap.Error(err))
			c.Status(http.StatusInternalServerError)
			return
		}

	case "customer.subscription.deleted":
		logger.Get().Info("Customer subscription deleted", zap.String("event_type", string(event.Type)), zap.String("event_id", event.ID))

		var subscription stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			logger.Get().Error("Error parsing subscription", zap.Error(err))
			c.Status(http.StatusBadRequest)
			return
		}

		logger.Get().Info("Parsed subscription", zap.String("customer_id", subscription.Customer.ID))

		stripeID = subscription.Customer.ID
		if err := db.UpdateStatusByStripeID(stripeID, models.UserStatusInactive, nil); err != nil {
			logger.Get().Error("Error updating user status", zap.Error(err))
			c.Status(http.StatusInternalServerError)
			return
		}

	default:
		logger.Get().Info("Unhandled event type", zap.String("event_type", string(event.Type)))
	}

	c.Status(http.StatusOK)
}

func HandleGetUser(c *gin.Context) {
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

	user_info, err := db.GetUserByID(claims.Sub)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user_info)
}

func HandleDeleteSubscription(c *gin.Context) {
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

	user_data, err := db.GetUserByID(claims.Sub)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	params := &stripe.SubscriptionCancelParams{}
	result, err := subscription.Cancel(*user_data.SubscriptionID, params)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error communicating with stripe upon cancellation": err.Error()})
	}

	accessTokens, err := db.DeletePlaidItemsByUserID(claims.Sub)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error deleting plaid items from postgres": err.Error()})
	}

	err = DeletePlaidItems(c, accessTokens)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error removing plaid items from plaid": err.Error()})
	}

	c.JSON(http.StatusOK, result)
}
