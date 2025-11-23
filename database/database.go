package repository

import (
	"fmt"
	"log"

	"Backend-trainee-assignment/config"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func NewDB(cfg *config.Config) (*sqlx.DB, error) {
	var connStr string
	if cfg.DatabaseURL != "" {
		connStr = cfg.DatabaseURL
	} else {
		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	}

	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка при подключении: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка при подключении: %w", err)
	}

	log.Println("Подключение к базе данных успешно")
	return db, nil
}
