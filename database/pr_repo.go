package repository

import (
	"Backend-trainee-assignment/models"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

type PRRepository struct {
	db *sqlx.DB
}

func (r *PRRepository) ReplacePRReviewer(prID string, oldReviewerID string, d string) any {
	panic("unimplemented")
}

func NewPRRepository(db *sqlx.DB) *PRRepository {
	return &PRRepository{db: db}
}

// Функция возвращает подключение к БД (нужно для stats handler)
func (r *PRRepository) DB() *sql.DB {
	return r.db.DB
}

// Функция создает новый Pull Request
func (r *PRRepository) CreatePR(pr *models.PullRequest) error {
	query := `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at, merged_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(query, pr.PullRequestID, pr.PullRequestName, pr.AuthorID, pr.Status, pr.CreatedAt, pr.MergedAt)
	return err
}

// Функция возвращает PR по ID
func (r *PRRepository) GetPRByID(prID string) (*models.PullRequest, error) {
	query := `
		SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		FROM pull_requests
		WHERE pull_request_id = $1
	`
	var pr models.PullRequest
	err := r.db.QueryRow(query, prID).Scan(
		&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status, &pr.CreatedAt, &pr.MergedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &pr, err
}

// Функция проверяет существование PR
func (r *PRRepository) PRExists(prID string) (bool, error) {
	pr, err := r.GetPRByID(prID)
	if err != nil {
		return false, err
	}
	return pr != nil, nil
}

// Функция добавляет ревьюера к PR
func (r *PRRepository) AddPRReviewer(prID, reviewerID string) error {
	query := `
		INSERT INTO pr_reviewers (pr_id, reviewer_user_id, assigned_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (pr_id, reviewer_user_id) DO NOTHING
	`
	_, err := r.db.Exec(query, prID, reviewerID, time.Now())
	return err
}

// Функция удаляет ревьюера из PR
func (r *PRRepository) RemovePRReviewer(prID, reviewerID string) error {
	query := `
		DELETE FROM pr_reviewers 
		WHERE pr_id = $1 AND reviewer_user_id = $2
	`
	_, err := r.db.Exec(query, prID, reviewerID)
	return err
}

// Функция возвращает ревьюеров PR
func (r *PRRepository) GetPRReviewers(prID string) ([]string, error) {
	query := `
		SELECT reviewer_user_id
		FROM pr_reviewers
		WHERE pr_id = $1
		ORDER BY assigned_at
	`
	rows, err := r.db.Query(query, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var reviewerID string
		err := rows.Scan(&reviewerID)
		if err != nil {
			return nil, err
		}
		reviewers = append(reviewers, reviewerID)
	}

	return reviewers, nil
}

// Функция возвращает PR вместе с ревьюерами
func (r *PRRepository) GetPRWithReviewers(prID string) (*models.PullRequest, []string, error) {
	pr, err := r.GetPRByID(prID)
	if err != nil {
		return nil, nil, err
	}
	if pr == nil {
		return nil, nil, nil
	}

	reviewers, err := r.GetPRReviewers(prID)
	if err != nil {
		return nil, nil, err
	}
	pr.AssignedReviewers = reviewers
	return pr, reviewers, nil
}

// Функция меняет статус PR
func (r *PRRepository) UpdatePRStatus(prID, status string, mergedAt *time.Time) error {
	query := `
		UPDATE pull_requests 
		SET status = $1, merged_at = $2
		WHERE pull_request_id = $3
	`
	_, err := r.db.Exec(query, status, mergedAt, prID)
	return err
}

// Функция возвращает PR назначенные пользователю
func (r *PRRepository) GetPRsByReviewer(reviewerID string) ([]models.PullRequestShort, error) {
	query := `
		SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status
		FROM pull_requests pr
		INNER JOIN pr_reviewers prv ON pr.pull_request_id = prv.pr_id
		WHERE prv.reviewer_user_id = $1 AND pr.status = 'OPEN'
		ORDER BY pr.created_at DESC
	`
	rows, err := r.db.Query(query, reviewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []models.PullRequestShort
	for rows.Next() {
		var pr models.PullRequestShort
		err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status)
		if err != nil {
			return nil, err
		}
		prs = append(prs, pr)
	}

	return prs, nil
}

func (r *PRRepository) GetOpenPRsByReviewer(reviewerID string) ([]models.PullRequest, error) {
	query := `
		SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status
		FROM pull_requests pr
		INNER JOIN pr_reviewers prv ON pr.pull_request_id = prv.pr_id
		WHERE prv.reviewer_user_id = $1 AND pr.status = 'OPEN'
	`

	rows, err := r.db.Query(query, reviewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []models.PullRequest
	for rows.Next() {
		var pr models.PullRequest
		err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status)
		if err != nil {
			return nil, err
		}
		prs = append(prs, pr)
	}

	return prs, nil
}
