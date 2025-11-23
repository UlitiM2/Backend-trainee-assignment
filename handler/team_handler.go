package handler

import (
	repository "Backend-trainee-assignment/database"
	"Backend-trainee-assignment/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

type TeamHandler struct {
	teamRepo *repository.TeamRepository
	userRepo *repository.UserRepository
}

func NewTeamHandler(teamRepo *repository.TeamRepository, userRepo *repository.UserRepository) *TeamHandler {
	return &TeamHandler{
		teamRepo: teamRepo,
		userRepo: userRepo,
	}
}

// Функция создает команду с участниками
func (h *TeamHandler) AddTeam(c *gin.Context) {
	var team models.Team
	if err := c.ShouldBindJSON(&team); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
		return
	}
	exists, err := h.teamRepo.TeamExists(team.TeamName)
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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "TEAM_EXISTS",
				"message": "team_name already exists",
			},
		})
		return
	}

	// Создание команды
	teamToCreate := &models.Team{
		TeamName: team.TeamName,
	}
	if err := h.teamRepo.CreateTeam(teamToCreate); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}
	for _, member := range team.Members {
		user := &models.User{
			UserID:   member.UserID,
			Username: member.Username,
			TeamName: team.TeamName,
			IsActive: member.IsActive,
		}
		if err := h.userRepo.CreateUser(user); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": map[string]interface{}{
					"code":    "INTERNAL_ERROR",
					"message": err.Error(),
				},
			})
			return
		}
	}

	createdTeam := models.Team{
		TeamName: team.TeamName,
		Members:  team.Members,
	}
	c.JSON(http.StatusCreated, gin.H{"team": createdTeam})
}

// Функция возвращает команду с участниками
func (h *TeamHandler) GetTeam(c *gin.Context) {
	teamName := c.Query("team_name")
	if teamName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]interface{}{
				"code":    "VALIDATION_ERROR",
				"message": "team_name is required",
			},
		})
		return
	}
	team, err := h.teamRepo.GetTeamByName(teamName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}
	if team == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]interface{}{
				"code":    "NOT_FOUND",
				"message": "team not found",
			},
		})
		return
	}

	members, err := h.userRepo.GetUsersByTeam(teamName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]interface{}{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	teamMembers := make([]models.TeamMember, len(members))
	for i, user := range members {
		teamMembers[i] = models.TeamMember{
			UserID:   user.UserID,
			Username: user.Username,
			IsActive: user.IsActive,
		}
	}
	teamWithMembers := models.Team{
		TeamName: team.TeamName,
		Members:  teamMembers,
	}
	c.JSON(http.StatusOK, teamWithMembers)
}
