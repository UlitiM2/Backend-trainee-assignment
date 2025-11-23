package handler

import (
	repository "Backend-trainee-assignment/database"
	"Backend-trainee-assignment/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userRepo *repository.UserRepository
	prRepo   *repository.PRRepository
}

func NewUserHandler(userRepo *repository.UserRepository, prRepo *repository.PRRepository) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
		prRepo:   prRepo,
	}
}

// Функция устанавливает флаг активности пользователя
func (h *UserHandler) SetIsActive(c *gin.Context) {
	var req struct {
		UserID   string `json:"user_id" binding:"required"`
		IsActive *bool  `json:"is_active" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
		return
	}
	user, err := h.userRepo.GetUserByID(req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]interface{}{
				"code":    "NOT_FOUND",
				"message": "user not found",
			},
		})
		return
	}

	// Обновление активности
	if err := h.userRepo.UpdateUserActiveStatus(req.UserID, *req.IsActive); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return

	}

	updatedUser, err := h.userRepo.GetUserByID(req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": updatedUser})
}

// Функция возвращает PR назначенные пользователю
func (h *UserHandler) GetReview(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "VALIDATION_ERROR",
				"message": "user_id is required",
			},
		})
		return
	}
	user, err := h.userRepo.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]interface{}{
				"code":    "NOT_FOUND",
				"message": "user not found",
			},
		})
		return
	}

	prs, err := h.prRepo.GetPRsByReviewer(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	var prShorts []models.PullRequestShort
	for _, pr := range prs {
		prShorts = append(prShorts, models.PullRequestShort{
			PullRequestID:   pr.PullRequestID,
			PullRequestName: pr.PullRequestName,
			AuthorID:        pr.AuthorID,
			Status:          pr.Status,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"user_id":       userID,
		"pull_requests": prShorts,
	})
}
