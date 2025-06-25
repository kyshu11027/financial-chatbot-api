package handlers

import (
	"finance-chatbot/api/middleware"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
)


var (
	successURL = "http://localhost:3000/"
	cancelURL  = "http://localhost:3000/canceled.html"
)

func HandleCreateStripeSession(c *gin.Context) {
	log.Printf("Price ID: %s", os.Getenv("STRIPE_PRICE_ID"))
	params := &stripe.CheckoutSessionParams{
		SuccessURL: &successURL,
		CancelURL:  &cancelURL,
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(os.Getenv("STRIPE_PRICE_ID")),
				Quantity: stripe.Int64(1),
			},
		},
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			TrialPeriodDays: stripe.Int64(1),
		},
	}

	s, _ := session.New(params)

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

	switch event.Type {
	case "checkout.session.completed":
		log.Printf("Checkout webhook received: %s", event.Type)
		// Update DB to save the customer_id

	case "customer.subscription.created":
		log.Printf("Customer subscription created webhook received: %s", event.Type)
		// Update DB to reflect subscription status is trialing.
		
	case "invoice.paid":
		log.Printf("Invoice paid webhook received: %s", event.Type)
		// Update DB to reflect subscription status is active.

	case "invoice.payment_failed":
		log.Printf("Invoice payment failed webhook received: %s", event.Type)
		// Update DB to reflect subscription status is inactive.
	default:
		log.Printf("Unhandled event type: %s", event.Type)
		// unhandled event type
	}
}
