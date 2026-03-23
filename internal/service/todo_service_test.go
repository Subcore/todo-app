package service

import (
	"context"
	"testing"
	"time"

	"github.com/Subcore/todo-app-v2/internal/model"
	"github.com/Subcore/todo-app-v2/internal/repository"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock of TodoRepository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, todo *model.Todo) error {
	args := m.Called(ctx, todo)
	return args.Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id uint) (*model.Todo, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Todo), args.Error(1)
}

func (m *MockRepository) GetAll(ctx context.Context, filter model.TodoFilter) ([]model.Todo, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]model.Todo), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, todo *model.Todo) error {
	args := m.Called(ctx, todo)
	return args.Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) DeleteCompleted(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockRepository) GetDeleted(ctx context.Context) ([]model.Todo, error) {
	args := m.Called(ctx)
	return args.Get(0).([]model.Todo), args.Error(1)
}

// Проверяем создание задачи: обычная, с пустым заголовком, с тегами и датой
func TestTodoService_CreateTodo(t *testing.T) {
	ctx := context.Background()

	// Успешное создание задачи с нормальным заголовком
	t.Run("success", func(t *testing.T) {
		// Подставляем фейковый репозиторий вместо настоящей базы
		mockRepo := new(MockRepository)
		svc := NewTodoService(mockRepo)

		title := "Test Todo"
		// Говорим фейку: когда попросят создать задачу с этим заголовком — верни nil (ошибок нет)
		mockRepo.On("Create", ctx, mock.MatchedBy(func(todo *model.Todo) bool {
			return todo.Title == title && !todo.Completed && len(todo.Tags) == 0
		})).Return(nil)

		todo, err := svc.CreateTodo(ctx, title, nil, nil)

		assert.NoError(t, err)
		assert.NotNil(t, todo)
		assert.Equal(t, title, todo.Title)
		assert.False(t, todo.Completed)
		assert.Equal(t, pq.StringArray([]string{}), todo.Tags)
		// Проверяем что фейк вызвали именно так, как договаривались
		mockRepo.AssertExpectations(t)
	})

	// Заголовок из одних пробелов должен отклоняться до обращения к базе
	t.Run("empty title error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewTodoService(mockRepo)

		todo, err := svc.CreateTodo(ctx, "   ", nil, nil)

		assert.ErrorIs(t, err, ErrEmptyTitle)
		assert.Nil(t, todo)
	})

	// Теги и дата переданные снаружи должны сохраниться в задаче
	t.Run("with tags and due date", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewTodoService(mockRepo)

		title := "Scoped Todo"
		tags := []string{"urgent", "work"}
		now := time.Now()

		// Говорим фейку: принять задачу только если у неё ровно 2 тега и есть дата
		mockRepo.On("Create", ctx, mock.MatchedBy(func(todo *model.Todo) bool {
			return todo.Title == title && len(todo.Tags) == 2 && todo.DueDate != nil
		})).Return(nil)

		todo, err := svc.CreateTodo(ctx, title, &now, tags)

		assert.NoError(t, err)
		assert.Equal(t, pq.StringArray(tags), todo.Tags)
		assert.Equal(t, &now, todo.DueDate)
		mockRepo.AssertExpectations(t)
	})
}

// Проверяем обновление задачи: успех, задача не найдена, пустой заголовок
func TestTodoService_UpdateTodo(t *testing.T) {
	ctx := context.Background()

	// Успешное обновление заголовка и статуса выполнения
	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewTodoService(mockRepo)

		trueVal := true
		existingTodo := &model.Todo{ID: 1, Title: "Old Title", Completed: false, Tags: []string{}}
		// Говорим фейку: при запросе задачи с ID=1 — вернуть существующую задачу
		mockRepo.On("GetByID", ctx, uint(1)).Return(existingTodo, nil)
		// Говорим фейку: принять обновление если изменились заголовок и статус
		mockRepo.On("Update", ctx, mock.MatchedBy(func(todo *model.Todo) bool {
			return todo.ID == 1 && todo.Title == "New Title" && todo.Completed == true
		})).Return(nil)

		todo, err := svc.UpdateTodo(ctx, 1, "New Title", &trueVal, nil, nil)

		assert.NoError(t, err)
		assert.Equal(t, "New Title", todo.Title)
		assert.True(t, todo.Completed)
		mockRepo.AssertExpectations(t)
	})

	// Обновление несуществующей задачи должно вернуть ErrNotFound
	t.Run("not found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewTodoService(mockRepo)

		// Говорим фейку: задача 999 не существует
		mockRepo.On("GetByID", ctx, uint(999)).Return(nil, repository.ErrNotFound)

		todo, err := svc.UpdateTodo(ctx, 999, "Title", nil, nil, nil)

		assert.ErrorIs(t, err, repository.ErrNotFound)
		assert.Nil(t, todo)
	})

	// Попытка установить пустой заголовок при обновлении должна отклоняться
	t.Run("empty title on update", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewTodoService(mockRepo)

		mockRepo.On("GetByID", ctx, uint(1)).Return(&model.Todo{ID: 1}, nil)

		todo, err := svc.UpdateTodo(ctx, 1, "", nil, nil, nil)

		assert.ErrorIs(t, err, ErrEmptyTitle)
		assert.Nil(t, todo)
	})
}

// Проверяем удаление задачи: успех и задача не найдена
func TestTodoService_DeleteTodo(t *testing.T) {
	ctx := context.Background()

	// Успешное удаление существующей задачи
	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewTodoService(mockRepo)

		// Говорим фейку: когда попросят удалить задачу 1 — ответить что всё ок
		mockRepo.On("Delete", ctx, uint(1)).Return(nil)

		err := svc.DeleteTodo(ctx, 1)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	// Удаление несуществующей задачи должно вернуть ErrNotFound
	t.Run("not found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewTodoService(mockRepo)

		// Говорим фейку: задача 999 не существует
		mockRepo.On("Delete", ctx, uint(999)).Return(repository.ErrNotFound)

		err := svc.DeleteTodo(ctx, 999)

		assert.ErrorIs(t, err, repository.ErrNotFound)
		mockRepo.AssertExpectations(t)
	})
}

// Проверяем что GetTodo и GetAllTodos просто передают данные из репозитория наверх без изменений
func TestTodoService_GetMethods(t *testing.T) {
	ctx := context.Background()

	// Получение одной задачи по ID
	t.Run("GetTodo", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewTodoService(mockRepo)
		expected := &model.Todo{ID: 1, Title: "Test"}
		// Говорим фейку: при запросе ID=1 вернуть заготовленную задачу
		mockRepo.On("GetByID", ctx, uint(1)).Return(expected, nil)

		todo, err := svc.GetTodo(ctx, 1)

		assert.NoError(t, err)
		assert.Equal(t, expected, todo)
	})

	// Получение всего списка задач без фильтров
	t.Run("GetAllTodos", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewTodoService(mockRepo)
		expected := []model.Todo{{ID: 1, Title: "Test"}}
		filter := model.TodoFilter{}
		// Говорим фейку: вернуть список из одной задачи при пустом фильтре
		mockRepo.On("GetAll", ctx, filter).Return(expected, nil)

		todos, err := svc.GetAllTodos(ctx, filter)

		assert.NoError(t, err)
		assert.Equal(t, expected, todos)
	})
}

// Проверяем что новые теги и дата сохраняются при обновлении задачи
func TestTodoService_UpdateTodo_WithTagsAndDueDate(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewTodoService(mockRepo)

	existingTodo := &model.Todo{ID: 5, Title: "Old", Tags: pq.StringArray{}, DueDate: nil}
	newTags := []string{"work", "urgent"}
	due := time.Now().Add(24 * time.Hour)

	// Говорим фейку: вернуть задачу без тегов и даты
	mockRepo.On("GetByID", ctx, uint(5)).Return(existingTodo, nil)
	// Говорим фейку: принять обновление только если появились 2 тега и дата
	mockRepo.On("Update", ctx, mock.MatchedBy(func(todo *model.Todo) bool {
		return todo.ID == 5 && len(todo.Tags) == 2 && todo.DueDate != nil
	})).Return(nil)

	todo, err := svc.UpdateTodo(ctx, 5, "New", nil, &due, newTags)

	assert.NoError(t, err)
	assert.Equal(t, pq.StringArray(newTags), todo.Tags)
	assert.Equal(t, &due, todo.DueDate)
	mockRepo.AssertExpectations(t)
}

// Проверяем что если передать nil вместо тегов — существующие теги остаются нетронутыми
func TestTodoService_UpdateTodo_NilTagsPreserveExisting(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewTodoService(mockRepo)

	existingTags := pq.StringArray{"work", "urgent"}
	existingTodo := &model.Todo{ID: 7, Title: "Has Tags", Tags: existingTags}

	// Говорим фейку: вернуть задачу с двумя тегами
	mockRepo.On("GetByID", ctx, uint(7)).Return(existingTodo, nil)
	// Говорим фейку: принять обновление — теги должны остаться (всё те же 2)
	mockRepo.On("Update", ctx, mock.MatchedBy(func(todo *model.Todo) bool {
		return todo.ID == 7 && len(todo.Tags) == 2
	})).Return(nil)

	// tags=nil → tags should remain ["work", "urgent"]
	todo, err := svc.UpdateTodo(ctx, 7, "Has Tags", nil, nil, nil)

	assert.NoError(t, err)
	assert.Equal(t, existingTags, todo.Tags)
	mockRepo.AssertExpectations(t)
}

// Проверяем что сервис обрезает пробелы вокруг заголовка перед сохранением
func TestTodoService_CreateTodo_TitleTrimmed(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewTodoService(mockRepo)

	// Говорим фейку: принять задачу только если заголовок уже без лишних пробелов
	mockRepo.On("Create", ctx, mock.MatchedBy(func(todo *model.Todo) bool {
		return todo.Title == "spaced title"
	})).Return(nil)

	todo, err := svc.CreateTodo(ctx, "  spaced title  ", nil, nil)

	assert.NoError(t, err)
	assert.Equal(t, "spaced title", todo.Title)
	mockRepo.AssertExpectations(t)
}

// Проверяем что сервис вызывает DeleteCompleted в репозитории без лишних действий
func TestTodoService_DeleteCompletedTodos(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewTodoService(mockRepo)

	// Говорим фейку: когда попросят удалить выполненные — ответить что всё ок
	mockRepo.On("DeleteCompleted", ctx).Return(nil)

	err := svc.DeleteCompletedTodos(ctx)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// Проверяем что сервис возвращает список удалённых задач из репозитория
func TestTodoService_GetDeletedTodos(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewTodoService(mockRepo)

	expected := []model.Todo{
		{ID: 10, Title: "Deleted 1"},
		{ID: 11, Title: "Deleted 2"},
	}
	// Говорим фейку: вернуть две удалённые задачи
	mockRepo.On("GetDeleted", ctx).Return(expected, nil)

	todos, err := svc.GetDeletedTodos(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expected, todos)
	mockRepo.AssertExpectations(t)
}
