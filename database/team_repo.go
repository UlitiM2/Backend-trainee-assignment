package repository

import (
	"Backend-trainee-assignment/models"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

type TeamRepository struct {
	db *sqlx.DB
}

func NewTeamRepository(db *sqlx.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

// Функция создает новую команду
func (r *TeamRepository) CreateTeam(team *models.Team) error {
	query := `
		INSERT INTO teams (team_name)
		VALUES ($1)
	`
	_, err := r.db.Exec(query, team.TeamName)
	return err
}

// Функция возвращает команду по имени
func (r *TeamRepository) GetTeamByName(teamName string) (*models.Team, error) {
	query := `
		SELECT team_name
		FROM teams
		WHERE team_name = $1
	`
	var team models.Team
	err := r.db.QueryRow(query, teamName).Scan(&team.TeamName)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &team, err
}

// Функция проверяет существование команды
func (r *TeamRepository) TeamExists(teamName string) (bool, error) {
	team, err := r.GetTeamByName(teamName)
	if err != nil {
		return false, err
	}
	return team != nil, nil
}

// Функция возвращает все команды
func (r *TeamRepository) GetAllTeams() ([]models.Team, error) {
	query := `
		SELECT team_name
		FROM teams
		ORDER BY team_name
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []models.Team
	for rows.Next() {
		var team models.Team
		err := rows.Scan(&team.TeamName)
		if err != nil {
			return nil, err
		}
		teams = append(teams, team)
	}

	return teams, nil
}
