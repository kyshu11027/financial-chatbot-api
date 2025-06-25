package handlers

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/webhook"
)


var (
	successURL = "http://localhost:3000/session_id={CHECKOUT_SESSION_ID}"
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
	}

	s, _ := session.New(params)

	c.JSON(http.StatusOK, gin.H{
		"url": s.URL,
	})
}

func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	b, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("io.ReadAll: %v", err)
		return
	}

	event, err := webhook.ConstructEvent(b, r.Header.Get("Stripe-Signature"), "{{STRIPE_WEBHOOK_SECRET}}")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("webhook.ConstructEvent: %v", err)
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		log.Printf("Checkout webhook received: %s", event.Type)
	case "invoice.paid":
		log.Printf("Invoice paid webhook received: %s", event.Type)
		// Continue to provision the subscription as payments continue to be made.
		// Store the status in your database and check when a user accesses your service.
		// This approach helps you avoid hitting rate limits.
	case "invoice.payment_failed":
		log.Printf("Invoice payment failed webhook received: %s", event.Type)
		// The payment failed or the customer does not have a valid payment method.
		// The subscription becomes past_due. Notify your customer and send them to the
		// customer portal to update their payment information.
	default:
		log.Printf("Unhandled event type: %s", event.Type)
		// unhandled event type
	}
}
