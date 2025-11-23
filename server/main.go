package main

import (
	"Backend-trainee-assignment/config"
	repository "Backend-trainee-assignment/database"
	"Backend-trainee-assignment/handler"
	service "Backend-trainee-assignment/services"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	cfg := config.Load()

	// Подключение БД
	db, err := repository.NewDB(cfg)
	if err != nil {
		log.Fatalf("Ошибка: %v", err)
	}
	defer db.Close()

	// Инициализация
	userRepo := repository.NewUserRepository(db)
	teamRepo := repository.NewTeamRepository(db)
	prRepo := repository.NewPRRepository(db)

	reviewerService := service.NewReviewerService(userRepo, teamRepo, prRepo)

	teamHandler := handler.NewTeamHandler(teamRepo, userRepo)
	userHandler := handler.NewUserHandler(userRepo, prRepo)
	prHandler := handler.NewPRHandler(prRepo, userRepo, reviewerService)
	statsHandler := handler.NewStatsHandler(prRepo)
	delHandler := handler.NewBulkHandler(userRepo, prRepo)
	router := gin.Default()

	// Openapi
	router.POST("/team/add", teamHandler.AddTeam)
	router.GET("/team/get", teamHandler.GetTeam)
	router.POST("/users/setIsActive", userHandler.SetIsActive)
	router.GET("/users/getReview", userHandler.GetReview)
	router.POST("/pullRequest/create", prHandler.CreatePR)
	router.POST("/pullRequest/merge", prHandler.MergePR)
	router.POST("/pullRequest/reassign", prHandler.ReassignReviewer)
	router.POST("/api/pull-requests/:id/reviewers", prHandler.AddReviewer)
	router.GET("/api/stats", statsHandler.GetStats)
	router.POST("/team/massDeactivate", delHandler.BulkDeactivate)

	log.Printf("Сервер запущен, порт: %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Ошибка при запуске: %v", err)
	}

}
