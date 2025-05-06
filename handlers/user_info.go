package handlers

import (
	"finance-chatbot/api/db"
	"finance-chatbot/api/middleware"
	"finance-chatbot/api/models"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CreateUserInfoRequest struct {
	Name        string  `json:"name"`
	Income      float64 `json:"income"`
	SavingsGoal float64 `json:"savings_goal"`
}

type UpdateUserInfoRequest struct {
	CreateUserInfoRequest
}

func CreateUserInfo(c *gin.Context) {
	var req CreateUserInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Error binding JSON for CreateUserInfo: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := c.Get("user")
	if !exists {
		log.Println("User not authenticated in CreateUserInfo")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	claims, ok := user.(*middleware.SupabaseClaims)
	if !ok {
		log.Println("Invalid user claims in CreateUserInfo")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user claims"})
		return
	}

	userInfo := &models.UserInfo{
		UserID:      claims.Sub,
		Name: 	  	 req.Name,
		Income:      req.Income,
		SavingsGoal: req.SavingsGoal,
	}

	err := db.CreateUserInfo(c, userInfo)
	if err != nil {
		log.Printf("Error creating user info for userID %s: %v", claims.Sub, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("User info created successfully for userID: %s", claims.Sub)
	c.JSON(http.StatusOK, gin.H{"message": "User info created successfully"})
}

func UpdateUserInfo(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		log.Println("User not authenticated in UpdateUserInfo")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	claims, ok := user.(*middleware.SupabaseClaims)
	if !ok {
		log.Println("Invalid user claims in UpdateUserInfo")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user claims"})
		return
	}

	var req UpdateUserInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Error binding JSON for UpdateUserInfo: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userInfo := &models.UserInfo{
		UserID:      claims.Sub,
		Name:        req.Name,
		Income:      req.Income,
		SavingsGoal: req.SavingsGoal,
	}

	err := db.UpdateUserInfo(c, claims.Sub, userInfo)
	if err != nil {
		log.Printf("Error updating user info for userID %s: %v", claims.Sub, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("User info updated successfully for userID: %s", claims.Sub)
	c.JSON(http.StatusOK, gin.H{"message": "User info updated successfully"})
}

func DeleteUserInfo(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		log.Println("User not authenticated in DeleteUserInfo")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	claims, ok := user.(*middleware.SupabaseClaims)
	if !ok {
		log.Println("Invalid user claims in DeleteUserInfo")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user claims"})
		return
	}

	err := db.DeleteUserInfo(c, claims.Sub)
	if err != nil {
		log.Printf("Error deleting user info for userID %s: %v", claims.Sub, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("User info deleted successfully for userID: %s", claims.Sub)
	c.JSON(http.StatusOK, gin.H{"message": "User info deleted successfully"})
}

func GetUserInfo(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		log.Println("User not authenticated in GetUserInfo")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	claims, ok := user.(*middleware.SupabaseClaims)
	if !ok {
		log.Println("Invalid user claims in GetUserInfo")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user claims"})
		return
	}

	userInfo, err := db.GetUserInfo(c, claims.Sub)
	if err != nil {
		log.Printf("Error retrieving user info for userID %s: %v", claims.Sub, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if userInfo == nil {
		log.Printf("No user info found for userID: %s", claims.Sub)
		c.JSON(http.StatusOK, gin.H{"no_user_info": true})
		return
	}

	log.Printf("Successfully retrieved user info for userID: %s", claims.Sub)
	c.JSON(http.StatusOK, userInfo)
}
