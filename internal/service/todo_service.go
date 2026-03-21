package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Subcore/todo-app-v2/internal/model"
	"github.com/Subcore/todo-app-v2/internal/repository"
)

var (
	ErrEmptyTitle = errors.New("todo title cannot be empty")
	ErrNotFound   = repository.ErrNotFound
)

type TodoService interface {
	CreateTodo(ctx context.Context, title string, dueDate *time.Time, tags []string) (*model.Todo, error)
	GetTodo(ctx context.Context, id uint) (*model.Todo, error)
	GetAllTodos(ctx context.Context, filter model.TodoFilter) ([]model.Todo, error)
	UpdateTodo(ctx context.Context, id uint, title string, completed bool, dueDate *time.Time, tags []string) (*model.Todo, error)
	DeleteTodo(ctx context.Context, id uint) error
	DeleteCompletedTodos(ctx context.Context) error
	GetDeletedTodos(ctx context.Context) ([]model.Todo, error)
}

type todoService struct {
	repo repository.TodoRepository
}

func NewTodoService(repo repository.TodoRepository) TodoService {
	return &todoService{repo: repo}
}

func (s *todoService) CreateTodo(ctx context.Context, title string, dueDate *time.Time, tags []string) (*model.Todo, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrEmptyTitle
	}

	if tags == nil {
		tags = []string{}
	}

	todo := &model.Todo{
		Title:     title,
		Completed: false,
		DueDate:   dueDate,
		Tags:      tags,
	}

	if err := s.repo.Create(ctx, todo); err != nil {
		return nil, err
	}

	return todo, nil
}

func (s *todoService) GetTodo(ctx context.Context, id uint) (*model.Todo, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *todoService) GetAllTodos(ctx context.Context, filter model.TodoFilter) ([]model.Todo, error) {
	return s.repo.GetAll(ctx, filter)
}

func (s *todoService) UpdateTodo(ctx context.Context, id uint, title string, completed bool, dueDate *time.Time, tags []string) (*model.Todo, error) {
	todo, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrEmptyTitle
	}

	if tags == nil {
		tags = []string{}
	}

	todo.Title = title
	todo.Completed = completed
	todo.DueDate = dueDate
	todo.Tags = tags

	if err := s.repo.Update(ctx, todo); err != nil {
		return nil, err
	}

	return todo, nil
}

func (s *todoService) DeleteTodo(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

func (s *todoService) DeleteCompletedTodos(ctx context.Context) error {
	return s.repo.DeleteCompleted(ctx)
}

func (s *todoService) GetDeletedTodos(ctx context.Context) ([]model.Todo, error) {
	return s.repo.GetDeleted(ctx)
}
