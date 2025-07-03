package handlers

import (
	"finance-chatbot/api/logger"
	"finance-chatbot/api/models"
	"finance-chatbot/api/mongodb"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func CreateUserInfo(c *gin.Context) {
	var req models.UserInfo
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

	req.UserID = claims.Sub
	req.CreatedAt = time.Now().Unix()

	err := mongodb.CreateUserInfo(c.Request.Context(), &req)
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

	var req models.UserInfo
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Get().Error("error binding JSON", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.UserID = claims.Sub

	err := mongodb.ReplaceUserInfo(c.Request.Context(), claims.Sub, &req)
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

	err := mongodb.DeleteUserInfo(c.Request.Context(), claims.Sub)
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

	userInfo, err := mongodb.GetUserInfo(c.Request.Context(), claims.Sub)
	if err != nil {
		logger.Get().Error("error retrieving user info",
			zap.String("user_id", claims.Sub),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if userInfo == nil {
		logger.Get().Debug("no user info found",
			zap.String("user_id", claims.Sub))
		c.JSON(http.StatusOK, gin.H{"no_user_info": true})
		return
	}

	logger.Get().Info("user info retrieved successfully",
		zap.String("user_id", claims.Sub))
	c.JSON(http.StatusOK, userInfo)
}
