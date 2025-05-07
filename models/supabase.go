package models

import "github.com/golang-jwt/jwt/v5"
type SupabaseClaims struct {
	jwt.RegisteredClaims
	Email       string `json:"email"`
	Sub         string `json:"sub"`
	Role        string `json:"role"`
	AppMetadata struct {
		Provider string `json:"provider"`
	} `json:"app_metadata"`
}