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

func TestTodoService_CreateTodo(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewTodoService(mockRepo)

		title := "Test Todo"
		mockRepo.On("Create", ctx, mock.MatchedBy(func(todo *model.Todo) bool {
			return todo.Title == title && !todo.Completed && len(todo.Tags) == 0
		})).Return(nil)

		todo, err := svc.CreateTodo(ctx, title, nil, nil)

		assert.NoError(t, err)
		assert.NotNil(t, todo)
		assert.Equal(t, title, todo.Title)
		assert.False(t, todo.Completed)
		assert.Equal(t, pq.StringArray([]string{}), todo.Tags)
		mockRepo.AssertExpectations(t)
	})

	t.Run("empty title error", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewTodoService(mockRepo)

		todo, err := svc.CreateTodo(ctx, "   ", nil, nil)

		assert.ErrorIs(t, err, ErrEmptyTitle)
		assert.Nil(t, todo)
	})

	t.Run("with tags and due date", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewTodoService(mockRepo)

		title := "Scoped Todo"
		tags := []string{"urgent", "work"}
		now := time.Now()

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

func TestTodoService_UpdateTodo(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewTodoService(mockRepo)

		trueVal := true
		existingTodo := &model.Todo{ID: 1, Title: "Old Title", Completed: false, Tags: []string{}}
		mockRepo.On("GetByID", ctx, uint(1)).Return(existingTodo, nil)
		mockRepo.On("Update", ctx, mock.MatchedBy(func(todo *model.Todo) bool {
			return todo.ID == 1 && todo.Title == "New Title" && todo.Completed == true
		})).Return(nil)

		todo, err := svc.UpdateTodo(ctx, 1, "New Title", &trueVal, nil, nil)

		assert.NoError(t, err)
		assert.Equal(t, "New Title", todo.Title)
		assert.True(t, todo.Completed)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewTodoService(mockRepo)

		mockRepo.On("GetByID", ctx, uint(999)).Return(nil, repository.ErrNotFound)

		todo, err := svc.UpdateTodo(ctx, 999, "Title", nil, nil, nil)

		assert.ErrorIs(t, err, repository.ErrNotFound)
		assert.Nil(t, todo)
	})

	t.Run("empty title on update", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewTodoService(mockRepo)

		mockRepo.On("GetByID", ctx, uint(1)).Return(&model.Todo{ID: 1}, nil)

		todo, err := svc.UpdateTodo(ctx, 1, "", nil, nil, nil)

		assert.ErrorIs(t, err, ErrEmptyTitle)
		assert.Nil(t, todo)
	})
}

func TestTodoService_DeleteTodo(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewTodoService(mockRepo)

		mockRepo.On("Delete", ctx, uint(1)).Return(nil)

		err := svc.DeleteTodo(ctx, 1)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewTodoService(mockRepo)

		mockRepo.On("Delete", ctx, uint(999)).Return(repository.ErrNotFound)

		err := svc.DeleteTodo(ctx, 999)

		assert.ErrorIs(t, err, repository.ErrNotFound)
		mockRepo.AssertExpectations(t)
	})
}

func TestTodoService_GetMethods(t *testing.T) {
	ctx := context.Background()

	t.Run("GetTodo", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewTodoService(mockRepo)
		expected := &model.Todo{ID: 1, Title: "Test"}
		mockRepo.On("GetByID", ctx, uint(1)).Return(expected, nil)

		todo, err := svc.GetTodo(ctx, 1)

		assert.NoError(t, err)
		assert.Equal(t, expected, todo)
	})

	t.Run("GetAllTodos", func(t *testing.T) {
		mockRepo := new(MockRepository)
		svc := NewTodoService(mockRepo)
		expected := []model.Todo{{ID: 1, Title: "Test"}}
		filter := model.TodoFilter{}
		mockRepo.On("GetAll", ctx, filter).Return(expected, nil)

		todos, err := svc.GetAllTodos(ctx, filter)

		assert.NoError(t, err)
		assert.Equal(t, expected, todos)
	})
}

func TestTodoService_UpdateTodo_WithTagsAndDueDate(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewTodoService(mockRepo)

	existingTodo := &model.Todo{ID: 5, Title: "Old", Tags: pq.StringArray{}, DueDate: nil}
	newTags := []string{"work", "urgent"}
	due := time.Now().Add(24 * time.Hour)

	mockRepo.On("GetByID", ctx, uint(5)).Return(existingTodo, nil)
	mockRepo.On("Update", ctx, mock.MatchedBy(func(todo *model.Todo) bool {
		return todo.ID == 5 && len(todo.Tags) == 2 && todo.DueDate != nil
	})).Return(nil)

	todo, err := svc.UpdateTodo(ctx, 5, "New", nil, &due, newTags)

	assert.NoError(t, err)
	assert.Equal(t, pq.StringArray(newTags), todo.Tags)
	assert.Equal(t, &due, todo.DueDate)
	mockRepo.AssertExpectations(t)
}

func TestTodoService_UpdateTodo_NilTagsPreserveExisting(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewTodoService(mockRepo)

	existingTags := pq.StringArray{"work", "urgent"}
	existingTodo := &model.Todo{ID: 7, Title: "Has Tags", Tags: existingTags}

	mockRepo.On("GetByID", ctx, uint(7)).Return(existingTodo, nil)
	mockRepo.On("Update", ctx, mock.MatchedBy(func(todo *model.Todo) bool {
		return todo.ID == 7 && len(todo.Tags) == 2
	})).Return(nil)

	// tags=nil → tags should remain ["work", "urgent"]
	todo, err := svc.UpdateTodo(ctx, 7, "Has Tags", nil, nil, nil)

	assert.NoError(t, err)
	assert.Equal(t, existingTags, todo.Tags)
	mockRepo.AssertExpectations(t)
}

func TestTodoService_CreateTodo_TitleTrimmed(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewTodoService(mockRepo)

	mockRepo.On("Create", ctx, mock.MatchedBy(func(todo *model.Todo) bool {
		return todo.Title == "spaced title"
	})).Return(nil)

	todo, err := svc.CreateTodo(ctx, "  spaced title  ", nil, nil)

	assert.NoError(t, err)
	assert.Equal(t, "spaced title", todo.Title)
	mockRepo.AssertExpectations(t)
}

func TestTodoService_DeleteCompletedTodos(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewTodoService(mockRepo)

	mockRepo.On("DeleteCompleted", ctx).Return(nil)

	err := svc.DeleteCompletedTodos(ctx)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestTodoService_GetDeletedTodos(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	svc := NewTodoService(mockRepo)

	expected := []model.Todo{
		{ID: 10, Title: "Deleted 1"},
		{ID: 11, Title: "Deleted 2"},
	}
	mockRepo.On("GetDeleted", ctx).Return(expected, nil)

	todos, err := svc.GetDeletedTodos(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expected, todos)
	mockRepo.AssertExpectations(t)
}
