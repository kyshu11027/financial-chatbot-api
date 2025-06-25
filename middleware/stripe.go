package middleware

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v82/webhook"
)
const StripeEventKey = "stripe_event"

func StripeWebhookVerifier(c *gin.Context) {
	if c.Request.Method != "POST" {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Method not allowed"})
		return
	}
	b, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		log.Printf("io.ReadAll: %v", err)
		return
	}

	event, err := webhook.ConstructEvent(b, c.Request.Header.Get("Stripe-Signature"), os.Getenv("STRIPE_WEBHOOK_SECRET"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		log.Printf("webhook.ConstructEvent: %v", err)
		return
	}

	c.Set(StripeEventKey, event)
	c.Next()
}
