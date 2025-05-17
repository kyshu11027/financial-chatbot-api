package handlers

import (
	"finance-chatbot/api/db"
	"finance-chatbot/api/logger"
	"finance-chatbot/api/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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
		logger.Get().Error("error binding JSON", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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

	userInfo := &models.UserInfo{
		UserID:      claims.Sub,
		Name:        req.Name,
		Income:      req.Income,
		SavingsGoal: req.SavingsGoal,
	}

	err := db.CreateUserInfo(c, userInfo)
	if err != nil {
		logger.Get().Error("error creating user info",
			zap.String("user_id", claims.Sub),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Get().Info("user info created successfully",
		zap.String("user_id", claims.Sub))
	c.JSON(http.StatusOK, gin.H{"message": "User info created successfully"})
}

func UpdateUserInfo(c *gin.Context) {
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

	var req UpdateUserInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Get().Error("error binding JSON", zap.Error(err))
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
		logger.Get().Error("error updating user info",
			zap.String("user_id", claims.Sub),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Get().Info("user info updated successfully",
		zap.String("user_id", claims.Sub))
	c.JSON(http.StatusOK, gin.H{"message": "User info updated successfully"})
}

func DeleteUserInfo(c *gin.Context) {
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

	err := db.DeleteUserInfo(c, claims.Sub)
	if err != nil {
		logger.Get().Error("error deleting user info",
			zap.String("user_id", claims.Sub),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.Get().Info("user info deleted successfully",
		zap.String("user_id", claims.Sub))
	c.JSON(http.StatusOK, gin.H{"message": "User info deleted successfully"})
}

func GetUserInfo(c *gin.Context) {
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

	userInfo, err := db.GetUserInfo(c, claims.Sub)
	if err != nil {
		logger.Get().Error("error retrieving user info",
			zap.String("user_id", claims.Sub),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if userInfo == nil {
		logger.Get().Info("no user info found",
			zap.String("user_id", claims.Sub))
		c.JSON(http.StatusOK, gin.H{"no_user_info": true})
		return
	}

	logger.Get().Info("user info retrieved successfully",
		zap.String("user_id", claims.Sub))
	c.JSON(http.StatusOK, userInfo)
}
