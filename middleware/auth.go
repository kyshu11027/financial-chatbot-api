package middleware

import (
	"finance-chatbot/api/logger"
	"finance-chatbot/api/models"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// AuthMiddleware verifies JWT tokens in requests
func AuthMiddleware(c *gin.Context) {
	tokenString := extractToken(c.Request)
	if tokenString == "" {
		logger.Get().Error("missing or invalid token")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid token"})
		return
	}

	claims := &models.SupabaseClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
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
		logger.Get().Error("error parsing claims", zap.Error(err))
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
		return
	}

	if !token.Valid {
		logger.Get().Error("invalid token")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Verify issuer
	if claims.Issuer != os.Getenv("SUPABASE_URL")+"/auth/v1" {
		logger.Get().Error("invalid token issuer",
			zap.String("issuer", claims.Issuer))
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token issuer"})
		return
	}

	// Set the claims in the context
	c.Set("user", claims)
	c.Next()
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}
