package repository

import (
	"context"

	"github.com/Subcore/todo-app-v2/internal/model"

	"gorm.io/gorm"
)

type TodoRepository interface {
	Create(ctx context.Context, todo *model.Todo) error
	GetByID(ctx context.Context, id uint) (*model.Todo, error)
	GetAll(ctx context.Context, filter model.TodoFilter) ([]model.Todo, error)
	Update(ctx context.Context, todo *model.Todo) error
	Delete(ctx context.Context, id uint) error
	DeleteCompleted(ctx context.Context) error
	GetDeleted(ctx context.Context) ([]model.Todo, error)
}

type todoRepository struct {
	db *gorm.DB
}

func NewTodoRepository(db *gorm.DB) TodoRepository {
	return &todoRepository{db: db}
}

func (r *todoRepository) Create(ctx context.Context, todo *model.Todo) error {
	return r.db.WithContext(ctx).Create(todo).Error
}

func (r *todoRepository) GetByID(ctx context.Context, id uint) (*model.Todo, error) {
	var todo model.Todo
	if err := r.db.WithContext(ctx).First(&todo, id).Error; err != nil {
		return nil, err
	}
	return &todo, nil
}

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

	if err := query.Find(&todos).Error; err != nil {
		return nil, err
	}
	return todos, nil
}

func (r *todoRepository) Update(ctx context.Context, todo *model.Todo) error {
	return r.db.WithContext(ctx).Save(todo).Error
}

// Delete performs a soft delete of a todo item by its ID.
// This is a soft delete because the model.Todo struct contains gorm.DeletedAt.
func (r *todoRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Todo{}, id).Error
}

// DeleteCompleted performs a soft delete of all todo items marked as completed.
// This is a soft delete because the model.Todo struct contains gorm.DeletedAt.
func (r *todoRepository) DeleteCompleted(ctx context.Context) error {
	return r.db.WithContext(ctx).Where("completed = ?", true).Delete(&model.Todo{}).Error
}

func (r *todoRepository) GetDeleted(ctx context.Context) ([]model.Todo, error) {
	var todos []model.Todo
	err := r.db.WithContext(ctx).Unscoped().Where("deleted_at IS NOT NULL").Find(&todos).Error
	return todos, err
}
