package repository

import (
	"Backend-trainee-assignment/models"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Функция создает нового пользователя
func (r *UserRepository) CreateUser(user *models.User) error {
	query := `
		INSERT INTO users (user_id, username, team_name, is_active)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET
			username = EXCLUDED.username,
			team_name = EXCLUDED.team_name,
			is_active = EXCLUDED.is_active
	`
	_, err := r.db.Exec(query, user.UserID, user.Username, user.TeamName, user.IsActive)
	return err
}

// Функция возвращает пользователя по ID
func (r *UserRepository) GetUserByID(userID string) (*models.User, error) {
	query := `
		SELECT user_id, username, team_name, is_active
		FROM users
		WHERE user_id = $1
	`
	var user models.User
	err := r.db.QueryRow(query, userID).Scan(
		&user.UserID, &user.Username, &user.TeamName, &user.IsActive,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &user, err
}

// Функция обновляет статус пользователя
func (r *UserRepository) UpdateUserActiveStatus(userID string, isActive bool) error {
	query := `
		UPDATE users 
		SET is_active = $1
		WHERE user_id = $2
	`
	_, err := r.db.Exec(query, isActive, userID)
	return err
}

// Функция возвращает список активных пользователей
func (r *UserRepository) GetActiveUsers() ([]models.User, error) {
	query := `
		SELECT user_id, username, team_name, is_active
		FROM users
		WHERE is_active = true
		ORDER BY username
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// Функция возвращает участников команды
func (r *UserRepository) GetUsersByTeam(teamName string) ([]models.User, error) {
	query := `
		SELECT user_id, username, team_name, is_active
		FROM users
		WHERE team_name = $1
		ORDER BY username
	`
	rows, err := r.db.Query(query, teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *UserRepository) BulkDeactivateUsers(teamName string, userIDs []string) (int, error) {
	query := `
		UPDATE users 
		SET is_active = false 
		WHERE team_name = $1 AND user_id = ANY($2)
	`
	result, err := r.db.Exec(query, teamName, pq.Array(userIDs))
	if err != nil {
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected), nil
}
