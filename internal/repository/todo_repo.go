package repository

import (
	"context"
	"errors"

	"github.com/Subcore/todo-app/internal/model"

	"gorm.io/gorm"
)

// ErrNotFound is returned when the requested record does not exist.
var ErrNotFound = errors.New("record not found")

// TodoRepository is the persistence interface for todos.
type TodoRepository interface {
	Create(ctx context.Context, todo *model.Todo) error
	GetByID(ctx context.Context, id uint) (*model.Todo, error)
	GetAll(ctx context.Context, filter model.TodoFilter) ([]model.Todo, error)
	Update(ctx context.Context, todo *model.Todo) error
	Delete(ctx context.Context, id uint) error
	DeleteCompleted(ctx context.Context) error
	GetDeleted(ctx context.Context) ([]model.Todo, error)
}

// todoRepository is the GORM-backed implementation of TodoRepository.
type todoRepository struct {
	db *gorm.DB
}

// NewTodoRepository creates a new todo repository bound to the given DB handle.
func NewTodoRepository(db *gorm.DB) TodoRepository {
	return &todoRepository{db: db}
}

// Create persists a new todo.
func (r *todoRepository) Create(ctx context.Context, todo *model.Todo) error {
	return r.db.WithContext(ctx).Create(todo).Error
}

// GetByID returns a todo by its ID, or ErrNotFound when missing.
func (r *todoRepository) GetByID(ctx context.Context, id uint) (*model.Todo, error) {
	var todo model.Todo
	if err := r.db.WithContext(ctx).First(&todo, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &todo, nil
}

// GetAll returns todos filtered by completion status, search query, and date range.
func (r *todoRepository) GetAll(ctx context.Context, filter model.TodoFilter) ([]model.Todo, error) {
	var todos []model.Todo
	query := r.db.WithContext(ctx).Model(&model.Todo{})

	if filter.Completed != nil {
		query = query.Where("completed = ?", *filter.Completed)
	}
	if filter.Search != "" {
		query = query.Where("title ILIKE ?", "%"+filter.Search+"%")
	}
	if filter.DueBefore != nil {
		query = query.Where("due_date < ?", filter.DueBefore)
	}
	if filter.DueAfter != nil {
		query = query.Where("due_date > ?", filter.DueAfter)
	}

	if err := query.Order("created_at DESC").Find(&todos).Error; err != nil {
		return nil, err
	}
	return todos, nil
}

// Update saves changes to the todo.
func (r *todoRepository) Update(ctx context.Context, todo *model.Todo) error {
	return r.db.WithContext(ctx).Save(todo).Error
}

// Delete soft-deletes a todo by ID. Returns ErrNotFound if no row was affected.
func (r *todoRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&model.Todo{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteCompleted soft-deletes every todo whose completed flag is true.
func (r *todoRepository) DeleteCompleted(ctx context.Context) error {
	return r.db.WithContext(ctx).Where("completed = ?", true).Delete(&model.Todo{}).Error
}

// GetDeleted returns every soft-deleted todo.
func (r *todoRepository) GetDeleted(ctx context.Context) ([]model.Todo, error) {
	var todos []model.Todo
	err := r.db.WithContext(ctx).Unscoped().Where("deleted_at IS NOT NULL").Find(&todos).Error
	return todos, err
}
