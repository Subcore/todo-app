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
	// ErrEmptyTitle — ошибка при попытке создать/обновить задачу с пустым заголовком
	ErrEmptyTitle = errors.New("todo title cannot be empty")
	// ErrNotFound — алиас ошибки «запись не найдена» из репозитория
	ErrNotFound = repository.ErrNotFound
)

// TodoService — интерфейс бизнес-логики для работы с задачами
type TodoService interface {
	CreateTodo(ctx context.Context, title string, dueDate *time.Time, tags []string) (*model.Todo, error)
	GetTodo(ctx context.Context, id uint) (*model.Todo, error)
	GetAllTodos(ctx context.Context, filter model.TodoFilter) ([]model.Todo, error)
	UpdateTodo(ctx context.Context, id uint, title string, completed *bool, dueDate *time.Time, tags []string) (*model.Todo, error)
	DeleteTodo(ctx context.Context, id uint) error
	DeleteCompletedTodos(ctx context.Context) error
	GetDeletedTodos(ctx context.Context) ([]model.Todo, error)
}

// todoService — реализация TodoService
type todoService struct {
	repo repository.TodoRepository
}

// NewTodoService создаёт новый сервис задач с переданным репозиторием
func NewTodoService(repo repository.TodoRepository) TodoService {
	return &todoService{repo: repo}
}

// CreateTodo создаёт новую задачу, обрезая пробелы в заголовке и инициализируя теги
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

// GetTodo возвращает задачу по ID
func (s *todoService) GetTodo(ctx context.Context, id uint) (*model.Todo, error) {
	return s.repo.GetByID(ctx, id)
}

// GetAllTodos возвращает все задачи с применением фильтра
func (s *todoService) GetAllTodos(ctx context.Context, filter model.TodoFilter) ([]model.Todo, error) {
	return s.repo.GetAll(ctx, filter)
}

// UpdateTodo обновляет задачу: заголовок, статус выполнения, срок и теги
func (s *todoService) UpdateTodo(ctx context.Context, id uint, title string, completed *bool, dueDate *time.Time, tags []string) (*model.Todo, error) {
	todo, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrEmptyTitle
	}

	todo.Title = title

	if completed != nil {
		todo.Completed = *completed
	}

	if dueDate != nil {
		todo.DueDate = dueDate
	}

	if tags != nil {
		todo.Tags = tags
	}

	if err := s.repo.Update(ctx, todo); err != nil {
		return nil, err
	}

	return todo, nil
}

// DeleteTodo удаляет задачу по ID
func (s *todoService) DeleteTodo(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

// DeleteCompletedTodos удаляет все выполненные задачи
func (s *todoService) DeleteCompletedTodos(ctx context.Context) error {
	return s.repo.DeleteCompleted(ctx)
}

// GetDeletedTodos возвращает список мягко удалённых задач
func (s *todoService) GetDeletedTodos(ctx context.Context) ([]model.Todo, error) {
	return s.repo.GetDeleted(ctx)
}
