package repository

import (
	"context"
	"errors"

	"github.com/Subcore/todo-app-v2/internal/model"

	"gorm.io/gorm"
)

// ErrNotFound возвращается когда запрашиваемая запись не найдена
var ErrNotFound = errors.New("record not found")

// TodoRepository — интерфейс репозитория для работы с задачами в базе данных
type TodoRepository interface {
	Create(ctx context.Context, todo *model.Todo) error
	GetByID(ctx context.Context, id uint) (*model.Todo, error)
	GetAll(ctx context.Context, filter model.TodoFilter) ([]model.Todo, error)
	Update(ctx context.Context, todo *model.Todo) error
	Delete(ctx context.Context, id uint) error
	DeleteCompleted(ctx context.Context) error
	GetDeleted(ctx context.Context) ([]model.Todo, error)
}

// todoRepository — реализация TodoRepository через GORM
type todoRepository struct {
	db *gorm.DB
}

// NewTodoRepository создаёт новый репозиторий задач с подключением к БД
func NewTodoRepository(db *gorm.DB) TodoRepository {
	return &todoRepository{db: db}
}

// Create сохраняет новую задачу в базе данных
func (r *todoRepository) Create(ctx context.Context, todo *model.Todo) error {
	return r.db.WithContext(ctx).Create(todo).Error
}

// GetByID возвращает задачу по её ID или ErrNotFound если не найдена
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

// GetAll возвращает список задач с применением фильтров (статус, поиск, даты)
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

// Update сохраняет изменения задачи в базе данных
func (r *todoRepository) Update(ctx context.Context, todo *model.Todo) error {
	return r.db.WithContext(ctx).Save(todo).Error
}

// Delete выполняет мягкое удаление задачи по ID.
// Возвращает ErrNotFound если ни одна запись не была затронута.
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

// DeleteCompleted выполняет мягкое удаление всех задач со статусом «выполнено»
func (r *todoRepository) DeleteCompleted(ctx context.Context) error {
	return r.db.WithContext(ctx).Where("completed = ?", true).Delete(&model.Todo{}).Error
}

// GetDeleted возвращает список всех мягко удалённых задач
func (r *todoRepository) GetDeleted(ctx context.Context) ([]model.Todo, error) {
	var todos []model.Todo
	err := r.db.WithContext(ctx).Unscoped().Where("deleted_at IS NOT NULL").Find(&todos).Error
	return todos, err
}
