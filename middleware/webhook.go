package middleware

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"finance-chatbot/api/logger"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/plaid/plaid-go/v20/plaid"
	"go.uber.org/zap"
)

var cachedPlaidKey *plaid.JWKPublicKey

// PlaidWebhookVerifier ensures incoming Plaid webhooks are authentic
func PlaidWebhookVerifier(c *gin.Context) {
	// Read and restore body for handler
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Get().Error("failed to read request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		c.Abort()
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Extract JWT from Plaid-Verification header
	tokenString := c.GetHeader("Plaid-Verification")
	if tokenString == "" {
		logger.Get().Error("missing Plaid-Verification header")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing Plaid-Verification header"})
		c.Abort()
		return
	}

	// Parse token header without verifying
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		logger.Get().Error("failed to parse JWT", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid JWT"})
		c.Abort()
		return
	}

	// Ensure correct algorithm
	if token.Method.Alg() != "ES256" {
		logger.Get().Error("unexpected JWT signing algorithm",
			zap.String("alg", token.Method.Alg()))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unexpected signing algorithm"})
		c.Abort()
		return
	}

	// Extract `kid` from token
	kid, ok := token.Header["kid"].(string)
	if !ok || kid == "" {
		logger.Get().Error("missing kid in JWT header")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing key ID"})
		c.Abort()
		return
	}

	// Fetch verification key if not cached
	if cachedPlaidKey == nil || cachedPlaidKey.Kid != kid {
		publicKey, fetchErr := fetchPlaidKey(kid)
		if fetchErr != nil {
			logger.Get().Error("failed to fetch public key", zap.Error(fetchErr))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to fetch verification key"})
			c.Abort()
			return
		}
		cachedPlaidKey = &publicKey
	}

	// Construct ECDSA public key
	pubKey, err := buildPublicKey(cachedPlaidKey)
	if err != nil {
		logger.Get().Error("failed to construct public key", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid public key"})
		c.Abort()
		return
	}

	// Fully verify token using public key
	_, err = jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return pubKey, nil
	})

	if err != nil {
		logger.Get().Error("JWT verification failed", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid JWT signature"})
		c.Abort()
		return
	}

	// JWT is valid, continue
	c.Next()
}

// fetchPlaidKey retrieves the public key from Plaid
func fetchPlaidKey(kid string) (plaid.JWKPublicKey, error) {
	cfg := plaid.NewConfiguration()
	cfg.AddDefaultHeader("PLAID-CLIENT-ID", os.Getenv("PLAID_CLIENT_ID"))
	cfg.AddDefaultHeader("PLAID-SECRET", os.Getenv("PLAID_SECRET"))
	cfg.UseEnvironment(plaid.Sandbox) // Change to Development or Production as needed

	client := plaid.NewAPIClient(cfg)
	ctx := context.Background()

	req := plaid.NewWebhookVerificationKeyGetRequest(kid)
	resp, _, err := client.PlaidApi.WebhookVerificationKeyGet(ctx).
		WebhookVerificationKeyGetRequest(*req).Execute()

	if err != nil {
		return plaid.JWKPublicKey{}, err
	}

	return resp.GetKey(), nil
}

// buildPublicKey constructs an ECDSA public key from a JWK
func buildPublicKey(jwk *plaid.JWKPublicKey) (*ecdsa.PublicKey, error) {
	xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return nil, fmt.Errorf("invalid X coordinate: %w", err)
	}

	yBytes, err := base64.RawURLEncoding.DecodeString(jwk.Y)
	if err != nil {
		return nil, fmt.Errorf("invalid Y coordinate: %w", err)
	}

	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}, nil
}
