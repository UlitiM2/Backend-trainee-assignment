package handler

import (
	repository "Backend-trainee-assignment/database"
	"Backend-trainee-assignment/models"
	service "Backend-trainee-assignment/services"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type PRHandler struct {
	prRepo          *repository.PRRepository
	userRepo        *repository.UserRepository
	reviewerService *service.ReviewerService
}

func NewPRHandler(prRepo *repository.PRRepository, userRepo *repository.UserRepository, reviewerService *service.ReviewerService) *PRHandler {
	return &PRHandler{
		prRepo:          prRepo,
		userRepo:        userRepo,
		reviewerService: reviewerService,
	}
}

// Функция создает новый PR
func (h *PRHandler) CreatePR(c *gin.Context) {
	var req struct {
		PullRequestID   string `json:"pull_request_id" binding:"required"`
		PullRequestName string `json:"pull_request_name" binding:"required"`
		AuthorID        string `json:"author_id" binding:"required"`
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

	// Проверка что PR не существует
	exists, err := h.prRepo.PRExists(req.PullRequestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{
			"error": map[string]interface{}{
				"code":    "PR_EXISTS",
				"message": "PR id already exists",
			},
		})
		return
	}

	// Проверка, что автор существует
	author, err := h.userRepo.GetUserByID(req.AuthorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}
	if author == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]interface{}{
				"code":    "NOT_FOUND",
				"message": "author not found",
			},
		})
		return
	}

	// Создание PR
	pr := &models.PullRequest{
		PullRequestID:   req.PullRequestID,
		PullRequestName: req.PullRequestName,
		AuthorID:        req.AuthorID,
		Status:          "OPEN",
		CreatedAt:       time.Now(),
	}
	if err := h.prRepo.CreatePR(pr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	// Автоматическое назначение ревьюеров
	if err := h.reviewerService.AssignReviewers(pr); err != nil {
		reviewers, _ := h.prRepo.GetPRReviewers(pr.PullRequestID)
		pr.AssignedReviewers = reviewers
		c.JSON(http.StatusCreated, gin.H{
			"pr":      pr,
			"warning": "PR created but reviewers assignment failed: " + err.Error(),
		})
		return
	}

	prWithReviewers, reviewers, err := h.prRepo.GetPRWithReviewers(pr.PullRequestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	prWithReviewers.AssignedReviewers = reviewers
	c.JSON(http.StatusCreated, gin.H{"pr": prWithReviewers})
}

// Функция мержит PR
func (h *PRHandler) MergePR(c *gin.Context) {
	var req struct {
		PullRequestID string `json:"pull_request_id" binding:"required"`
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

	pr, reviewers, err := h.prRepo.GetPRWithReviewers(req.PullRequestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}
	if pr == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]interface{}{
				"code":    "NOT_FOUND",
				"message": "PR not found",
			},
		})
		return
	}
	// Проверка ситуации второго мерджа
	if pr.Status == "MERGED" {
		pr.AssignedReviewers = reviewers
		c.JSON(http.StatusOK, gin.H{"pr": pr})
		return
	}
	mergedAt := time.Now()
	if err := h.prRepo.UpdatePRStatus(req.PullRequestID, "MERGED", &mergedAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	mergedPR, reviewers, _ := h.prRepo.GetPRWithReviewers(req.PullRequestID)
	mergedPR.AssignedReviewers = reviewers
	mergedPR.MergedAt = &mergedAt
	c.JSON(http.StatusOK, gin.H{"pr": mergedPR})
}

// Функция переназначает ревьюера
func (h *PRHandler) ReassignReviewer(c *gin.Context) {
	var req struct {
		PullRequestID string `json:"pull_request_id" binding:"required"`
		OldUserID     string `json:"old_user_id" binding:"required"`
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

	newReviewerID, err := h.reviewerService.ReassignReviewer(req.PullRequestID, req.OldUserID)
	if err != nil {
		errorResponse := gin.H{
			"error": map[string]interface{}{
				"message": err.Error(),
			},
		}

		switch err.Error() {
		case "PR не найдено":
			errorResponse["error"].(map[string]interface{})["code"] = "NOT_FOUND"
			c.JSON(http.StatusNotFound, errorResponse)
		case "статус PR MERGED":
			errorResponse["error"].(map[string]interface{})["code"] = "PR_MERGED"
			c.JSON(http.StatusConflict, errorResponse)
		case "ревьюер не относится к данному PR":
			errorResponse["error"].(map[string]interface{})["code"] = "NOT_ASSIGNED"
			c.JSON(http.StatusConflict, errorResponse)
		case "нет доступных ревьюеров":
			errorResponse["error"].(map[string]interface{})["code"] = "NO_CANDIDATE"
			c.JSON(http.StatusConflict, errorResponse)
		default:
			errorResponse["error"].(map[string]interface{})["code"] = "INTERNAL_ERROR"
			c.JSON(http.StatusInternalServerError, errorResponse)
		}
		return
	}

	pr, reviewers, err := h.prRepo.GetPRWithReviewers(req.PullRequestID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	pr.AssignedReviewers = reviewers
	c.JSON(http.StatusOK, gin.H{
		"pr":          pr,
		"replaced_by": newReviewerID,
	})
}

// Функция добавляет ревьюера к PR
func (h *PRHandler) AddReviewer(c *gin.Context) {
	prID := c.Param("id")
	var req struct {
		ReviewerID string `json:"reviewer_id" binding:"required"`
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

	// Проверка, что PR существует и не замержен
	pr, reviewers, err := h.prRepo.GetPRWithReviewers(prID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}
	if pr == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]interface{}{
				"code":    "NOT_FOUND",
				"message": "PR not found",
			},
		})
		return
	}
	if pr.Status == "MERGED" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "PR_MERGED",
				"message": "cannot add reviewers to merged PR",
			},
		})
		return
	}

	// Проверка, что ревьюеров не больше 2
	if len(reviewers) >= 2 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "MAX_REVIEWERS",
				"message": "PR already has maximum reviewers (2)",
			},
		})
		return
	}

	// Проверка пользователя
	reviewer, err := h.userRepo.GetUserByID(req.ReviewerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}
	if reviewer == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]interface{}{
				"code":    "NOT_FOUND",
				"message": "reviewer not found",
			},
		})
		return
	}
	if !reviewer.IsActive {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "USER_INACTIVE",
				"message": "cannot assign inactive user as reviewer",
			},
		})
		return
	}
	if reviewer.UserID == pr.AuthorID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "AUTHOR_SELF_REVIEW",
				"message": "cannot assign author as reviewer",
			},
		})
		return
	}
	for _, existingReviewer := range reviewers {
		if existingReviewer == req.ReviewerID {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]interface{}{
					"code":    "ALREADY_ASSIGNED",
					"message": "user is already assigned as reviewer",
				},
			})
			return
		}
	}

	// Добавка ревьюера
	if err := h.prRepo.AddPRReviewer(prID, req.ReviewerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	updatedPR, updatedReviewers, _ := h.prRepo.GetPRWithReviewers(prID)
	updatedPR.AssignedReviewers = updatedReviewers
	c.JSON(http.StatusOK, gin.H{
		"message": "Reviewer added successfully",
		"pr":      updatedPR,
	})
}
