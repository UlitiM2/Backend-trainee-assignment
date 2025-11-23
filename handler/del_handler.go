package handler

import (
	repository "Backend-trainee-assignment/database"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type BulkHandler struct {
	userRepo *repository.UserRepository
	prRepo   *repository.PRRepository
}

func NewBulkHandler(userRepo *repository.UserRepository, prRepo *repository.PRRepository) *BulkHandler {
	return &BulkHandler{
		userRepo: userRepo,
		prRepo:   prRepo,
	}
}

type BulkDeactivateRequest struct {
	TeamName string   `json:"team_name" binding:"required"`
	UserIDs  []string `json:"user_ids" binding:"required"`
}

// Функция массово деактивирует пользователей и переназначает PR
func (h *BulkHandler) BulkDeactivate(c *gin.Context) {
	startTime := time.Now()
	var req BulkDeactivateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	// Деактивация пользователей
	deactivatedUsers, err := h.userRepo.BulkDeactivateUsers(req.TeamName, req.UserIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	// Переназначаение ревьюеров
	reassignedPRs, err := h.reassignReviewersForDeactivatedUsers(req.UserIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	processingTime := time.Since(startTime).Milliseconds()
	c.JSON(http.StatusOK, gin.H{
		"deactivated_users":  deactivatedUsers,
		"reassigned_prs":     reassignedPRs,
		"processing_time_ms": processingTime,
		"message":            "Bulk deactivation completed successfully",
	})
}

// Функция по переназначению
func (h *BulkHandler) reassignReviewersForDeactivatedUsers(userIDs []string) ([]string, error) {
	var reassignedPRs []string
	for _, userID := range userIDs {
		prs, err := h.prRepo.GetOpenPRsByReviewer(userID)
		if err != nil {
			return nil, err
		}

		for _, pr := range prs {
			newReviewer, err := h.findReplacementReviewer(pr.PullRequestID, userID)
			if err == nil && newReviewer != "" {
				// Удаление старого и добавление нового
				h.prRepo.RemovePRReviewer(pr.PullRequestID, userID)
				h.prRepo.AddPRReviewer(pr.PullRequestID, newReviewer)
				reassignedPRs = append(reassignedPRs, pr.PullRequestID)
			} else {
				h.prRepo.RemovePRReviewer(pr.PullRequestID, userID)
				reassignedPRs = append(reassignedPRs, pr.PullRequestID)
			}
		}
	}

	return reassignedPRs, nil
}

func (h *BulkHandler) findReplacementReviewer(prID, oldReviewerID string) (string, error) {
	pr, err := h.prRepo.GetPRByID(prID)
	if err != nil {
		return "", err
	}
	author, err := h.userRepo.GetUserByID(pr.AuthorID)
	if err != nil {
		return "", err
	}
	teamMembers, err := h.userRepo.GetUsersByTeam(author.TeamName)
	if err != nil {
		return "", err
	}
	currentReviewers, err := h.prRepo.GetPRReviewers(prID)
	if err != nil {
		return "", err
	}

	// Поиск замены
	for _, member := range teamMembers {
		if member.IsActive &&
			member.UserID != pr.AuthorID &&
			member.UserID != oldReviewerID &&
			!contains(currentReviewers, member.UserID) {
			return member.UserID, nil
		}
	}

	return "", nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
