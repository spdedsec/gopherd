package service

import (
	"context"
	"errors"
	"strings"

	"github.com/spdedsec/gopherd/internal/model"
	"github.com/spdedsec/gopherd/internal/repository"
)

type TaskService struct{ repo *repository.TaskRepository }

func NewTaskService(r *repository.TaskRepository) *TaskService { return &TaskService{repo: r} }
func (s *TaskService) Create(ctx context.Context, userID, title, description string) (model.Task, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	if title == "" {
		return model.Task{}, errors.New("title is required")
	}
	if len([]rune(title)) > 200 {
		return model.Task{}, errors.New("title must be 200 characters or fewer")
	}
	if len([]rune(description)) > 5000 {
		return model.Task{}, errors.New("description must be 5000 characters or fewer")
	}
	return s.repo.Create(ctx, userID, title, description)
}
func (s *TaskService) List(ctx context.Context, userID string, completed *bool, limit, offset int) ([]model.Task, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, userID, completed, limit, offset)
}
func (s *TaskService) Get(ctx context.Context, userID, id string) (model.Task, error) {
	return s.repo.Get(ctx, userID, id)
}
func (s *TaskService) Update(ctx context.Context, userID, id, title, description string, completed bool) (model.Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return model.Task{}, errors.New("title is required")
	}
	if len([]rune(title)) > 200 || len([]rune(description)) > 5000 {
		return model.Task{}, errors.New("field exceeds maximum length")
	}
	return s.repo.Update(ctx, userID, id, title, strings.TrimSpace(description), completed)
}
func (s *TaskService) Delete(ctx context.Context, userID, id string) error {
	return s.repo.Delete(ctx, userID, id)
}
