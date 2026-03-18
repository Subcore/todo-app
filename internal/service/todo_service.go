package service

import (
	"context"
	"errors"
	"strings"
	"github.com/Subcore/todo-app-v2/internal/model"
	"github.com/Subcore/todo-app-v2/internal/repository"
)

var (
	ErrEmptyTitle = errors.New("todo title cannot be empty")
	ErrNotFound   = errors.New("todo not found")
)

type TodoService interface {
	CreateTodo(ctx context.Context, title string) (*model.Todo, error)
	GetTodo(ctx context.Context, id uint) (*model.Todo, error)
	GetAllTodos(ctx context.Context) ([]model.Todo, error)
	UpdateTodo(ctx context.Context, id uint, title string, completed bool) (*model.Todo, error)
	DeleteTodo(ctx context.Context, id uint) error
}

type todoService struct {
	repo repository.TodoRepository
}

func NewTodoService(repo repository.TodoRepository) TodoService {
	return &todoService{repo: repo}
}

func (s *todoService) CreateTodo(ctx context.Context, title string) (*model.Todo, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrEmptyTitle
	}

	todo := &model.Todo{
		Title:     title,
		Completed: false,
	}

	if err := s.repo.Create(ctx, todo); err != nil {
		return nil, err
	}

	return todo, nil
}

func (s *todoService) GetTodo(ctx context.Context, id uint) (*model.Todo, error) {
	todo, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return todo, nil
}

func (s *todoService) GetAllTodos(ctx context.Context) ([]model.Todo, error) {
	return s.repo.GetAll(ctx)
}

func (s *todoService) UpdateTodo(ctx context.Context, id uint, title string, completed bool) (*model.Todo, error) {
	todo, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrEmptyTitle
	}

	todo.Title = title
	todo.Completed = completed

	if err := s.repo.Update(ctx, todo); err != nil {
		return nil, err
	}

	return todo, nil
}

func (s *todoService) DeleteTodo(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}
