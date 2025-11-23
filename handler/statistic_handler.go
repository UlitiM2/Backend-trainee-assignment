package handler

import (
	repository "Backend-trainee-assignment/database"
	"net/http"

	"github.com/gin-gonic/gin"
)

type StatsHandler struct {
	prRepo *repository.PRRepository
}

func NewStatsHandler(prRepo *repository.PRRepository) *StatsHandler {
	return &StatsHandler{prRepo: prRepo}
}

type StatsResponse struct {
	UserAssignments  map[string]int `json:"user_assignments"`
	PRAssignments    map[string]int `json:"pr_assignments"`
	TotalAssignments int            `json:"total_assignments"`
	ActivePRs        int            `json:"active_prs"`
	MergedPRs        int            `json:"merged_prs"`
}

// Функция возвращает статистику
func (h *StatsHandler) GetStats(c *gin.Context) {
	userStats, err := h.getUserAssignmentStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	prStats, totalAssignments, activePRs, mergedPRs, err := h.getPRAssignmentStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	response := StatsResponse{
		UserAssignments:  userStats,
		PRAssignments:    prStats,
		TotalAssignments: totalAssignments,
		ActivePRs:        activePRs,
		MergedPRs:        mergedPRs,
	}
	c.JSON(http.StatusOK, response)
}

// Функция возвращает статистику по пользователям
func (h *StatsHandler) getUserAssignmentStats() (map[string]int, error) {
	query := `
		SELECT reviewer_user_id, COUNT(*) as assignment_count
		FROM pr_reviewers
		GROUP BY reviewer_user_id
		ORDER BY assignment_count DESC
	`
	rows, err := h.prRepo.DB().Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var userID string
		var count int
		if err := rows.Scan(&userID, &count); err != nil {
			return nil, err
		}
		stats[userID] = count
	}
	return stats, nil
}

// Функция возвращает статистику по PR
func (h *StatsHandler) getPRAssignmentStats() (map[string]int, int, int, int, error) {
	prQuery := `
		SELECT pr_id, COUNT(*) as reviewer_count
		FROM pr_reviewers
		GROUP BY pr_id
	`
	prRows, err := h.prRepo.DB().Query(prQuery)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	defer prRows.Close()

	prStats := make(map[string]int)
	totalAssignments := 0
	for prRows.Next() {
		var prID string
		var count int
		if err := prRows.Scan(&prID, &count); err != nil {
			return nil, 0, 0, 0, err
		}
		prStats[prID] = count
		totalAssignments += count
	}

	statusQuery := `
		SELECT status, COUNT(*) 
		FROM pull_requests 
		GROUP BY status
	`
	statusRows, err := h.prRepo.DB().Query(statusQuery)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	defer statusRows.Close()

	activePRs, mergedPRs := 0, 0
	for statusRows.Next() {
		var status string
		var count int
		if err := statusRows.Scan(&status, &count); err != nil {
			return nil, 0, 0, 0, err
		}
		switch status {
		case "OPEN":
			activePRs = count
		case "MERGED":
			mergedPRs = count
		}
	}
	return prStats, totalAssignments, activePRs, mergedPRs, nil
}
