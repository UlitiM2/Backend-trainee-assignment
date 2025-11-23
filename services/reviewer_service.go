package service

import (
	repository "Backend-trainee-assignment/database"
	"Backend-trainee-assignment/models"
	"fmt"
	"math/rand"
	"time"
)

type ReviewerService struct {
	userRepo *repository.UserRepository
	teamRepo *repository.TeamRepository
	prRepo   *repository.PRRepository
}

func NewReviewerService(userRepo *repository.UserRepository, teamRepo *repository.TeamRepository, prRepo *repository.PRRepository) *ReviewerService {
	return &ReviewerService{
		userRepo: userRepo,
		teamRepo: teamRepo,
		prRepo:   prRepo,
	}
}

// Функция  назначает ревьюеров на PR
func (s *ReviewerService) AssignReviewers(pr *models.PullRequest) error {
	// Поиск автора и проверка ошибок
	author, err := s.userRepo.GetUserByID(pr.AuthorID)
	if err != nil {
		return fmt.Errorf("ошибка при поиске автора: %w", err)
	}
	if author == nil {
		return fmt.Errorf("автор не найден")
	}
	authorTeam := author.TeamName
	if authorTeam == "" {
		return fmt.Errorf("автор не относится к комнаде")
	}
	availableReviewers, err := s.getAvailableReviewers(authorTeam, pr.AuthorID)
	if err != nil {
		return fmt.Errorf("не удалось найти ревьюеров: %w", err)
	}
	if len(availableReviewers) == 0 {
		fmt.Printf("доступных ревьюеров нет")

		return nil
	}

	// Выбор радомных ревьюеров
	selectedReviewers := s.selectRandomReviewers(availableReviewers, 2)

	for _, reviewer := range selectedReviewers {
		if err := s.prRepo.AddPRReviewer(pr.PullRequestID, reviewer.UserID); err != nil {
			return fmt.Errorf("failed to assign reviewer: %w", err)
		}
	}

	return nil
}

// Функция заменяет ревьюера
func (s *ReviewerService) ReassignReviewer(prID, oldReviewerID string) (string, error) {
	// PR с ревьюерами и проверка ошибок
	pr, reviewers, err := s.prRepo.GetPRWithReviewers(prID)
	if err != nil {
		return "", fmt.Errorf("ошибка в получении PR: %w", err)
	}
	if pr == nil {
		return "", fmt.Errorf("PR не найден")
	}
	if pr.Status == "MERGED" {
		return "", fmt.Errorf("нельзя добавить ревьюера в MERGED PR")
	}

	// Поиск заменяемого ревьюера в PR
	isOldReviewerAssigned := false
	for _, reviewer := range reviewers {
		if reviewer == oldReviewerID {
			isOldReviewerAssigned = true
			break
		}
	}
	if !isOldReviewerAssigned {
		return "", fmt.Errorf("заменяемый ревьюер не назначен на PR")
	}

	// Поиск команды заменяемого ревьюера
	oldReviewer, err := s.userRepo.GetUserByID(oldReviewerID)
	if err != nil {
		return "", fmt.Errorf("ошибка в поиске ревьюера: %w", err)
	}
	if oldReviewer == nil {
		return "", fmt.Errorf("ревьюер не найден")
	}
	reviewerTeam := oldReviewer.TeamName
	if reviewerTeam == "" {
		return "", fmt.Errorf("ревьюер не относится к команде")
	}

	// Поиск доступных ревьюеров
	availableReviewers, err := s.getAvailableReviewersForReassignment(reviewerTeam, prID, oldReviewerID)
	if err != nil {
		return "", fmt.Errorf("ошибка при поиске доступных ревьюеров: %w", err)
	}
	if len(availableReviewers) == 0 {
		return "", fmt.Errorf("нет доступных ревьюеров")
	}

	// Замена на случайного ревьюера
	newReviewer := s.selectRandomReviewers(availableReviewers, 1)[0]
	if err := s.prRepo.RemovePRReviewer(prID, oldReviewerID); err != nil {
		return "", fmt.Errorf("ошибка при замене ревьюера: %w", err)
	}
	if err := s.prRepo.AddPRReviewer(prID, newReviewer.UserID); err != nil {
		s.prRepo.AddPRReviewer(prID, oldReviewerID)
		return "", fmt.Errorf("ошибка при добавлении нового ревьюера: %w", err)
	}

	currentReviewers, err := s.prRepo.GetPRReviewers(prID)
	if err != nil {
		return "", fmt.Errorf("ошибка при поиске доступных ревьюеров: %w", err)
	}
	if len(currentReviewers) == 1 {
		fmt.Printf("только один ревьюер\n")
		pr, err := s.prRepo.GetPRByID(prID)
		if err != nil {
			return newReviewer.UserID, nil
		}
		additionalReviewer, err := s.findAdditionalReviewer(pr, currentReviewers[0])
		if err == nil && additionalReviewer != nil {
			if err := s.prRepo.AddPRReviewer(prID, additionalReviewer.UserID); err != nil {
				fmt.Printf("ошибка в добавлении ревьюера: %v\n", err)
			} else {
				fmt.Printf("добавлен ревьюер: %s\n", additionalReviewer.UserID)
			}
		} else {
			fmt.Printf("нет доступных ревьюеров: %v\n", err)
		}
	}
	return newReviewer.UserID, nil
}

// Функция находит дополнительного ревьюера
func (s *ReviewerService) findAdditionalReviewer(pr *models.PullRequest, existingReviewer string) (*models.User, error) {
	author, err := s.userRepo.GetUserByID(pr.AuthorID)
	if err != nil {
		return nil, err
	}
	if author == nil {
		return nil, fmt.Errorf("author not found")
	}
	authorTeam := author.TeamName
	if authorTeam == "" {
		return nil, fmt.Errorf("author not in any team")
	}

	// Проверка критериев (активный, не автор, не исключаемый ревьюер)
	teamMembers, err := s.userRepo.GetUsersByTeam(authorTeam)
	if err != nil {
		return nil, err
	}
	var available []models.User
	for _, member := range teamMembers {
		if member.IsActive &&
			member.UserID != pr.AuthorID &&
			member.UserID != existingReviewer {
			available = append(available, member)
		}
	}
	if len(available) == 0 {
		return nil, fmt.Errorf("no available reviewers")
	}

	return &s.selectRandomReviewers(available, 1)[0], nil
}

// Функция возвращает доступных ревьюеров для переназначения
func (s *ReviewerService) getAvailableReviewersForReassignment(teamName, prID, excludeReviewerID string) ([]models.User, error) {
	// Поиск участников команды и проверка ошибок
	teamMembers, err := s.userRepo.GetUsersByTeam(teamName)
	if err != nil {
		return nil, err
	}
	pr, err := s.prRepo.GetPRByID(prID)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, fmt.Errorf("PR не найден")
	}

	// Проверка критериев (активный, не автор, не исключаемый ревьюер)
	var available []models.User
	for _, member := range teamMembers {
		switch {
		case !member.IsActive:
			fmt.Printf("✗ Excluded: %s (%s) - Reason: inactive\n", member.UserID, member.Username)
		case member.UserID == pr.AuthorID:
			fmt.Printf("✗ Excluded: %s (%s) - Reason: author\n", member.UserID, member.Username)
		case member.UserID == excludeReviewerID:
			fmt.Printf("✗ Excluded: %s (%s) - Reason: excluded\n", member.UserID, member.Username)
		default:
			available = append(available, member)
			fmt.Printf("✓ Available candidate: %s (%s)\n", member.UserID, member.Username)
		}
	}
	return available, nil
}

// Функция возвращает доступных ревьюеров при первом назначении
func (s *ReviewerService) getAvailableReviewers(teamName, authorID string) ([]models.User, error) {
	if teamName == "" {
		return []models.User{}, nil
	}
	teamMembers, err := s.userRepo.GetUsersByTeam(teamName)
	if err != nil {
		return nil, err
	}

	// Проверка критериев(активный, не автор)
	var available []models.User
	for _, member := range teamMembers {
		if member.IsActive && member.UserID != authorID {
			available = append(available, member)
		}
	}

	return available, nil
}

// Функция выбирает случайных ревьюеров
func (s *ReviewerService) selectRandomReviewers(reviewers []models.User, max int) []models.User {
	if len(reviewers) == 0 {
		return []models.User{}
	}
	if len(reviewers) <= max {
		return reviewers
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(reviewers), func(i, j int) {
		reviewers[i], reviewers[j] = reviewers[j], reviewers[i]
	})
	if max > len(reviewers) {
		max = len(reviewers)
	}

	return reviewers[:max]
}
