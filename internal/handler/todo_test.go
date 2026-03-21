package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Subcore/todo-app-v2/internal/model"
	"github.com/Subcore/todo-app-v2/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockTodoService implements service.TodoService using testify/mock
type MockTodoService struct {
	mock.Mock
}

func (m *MockTodoService) CreateTodo(ctx context.Context, title string, dueDate *time.Time, tags []string) (*model.Todo, error) {
	args := m.Called(ctx, title, dueDate, tags)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Todo), args.Error(1)
}

func (m *MockTodoService) GetTodo(ctx context.Context, id uint) (*model.Todo, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Todo), args.Error(1)
}

func (m *MockTodoService) GetAllTodos(ctx context.Context, filter model.TodoFilter) ([]model.Todo, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]model.Todo), args.Error(1)
}

func (m *MockTodoService) UpdateTodo(ctx context.Context, id uint, title string, completed bool, dueDate *time.Time, tags []string) (*model.Todo, error) {
	args := m.Called(ctx, id, title, completed, dueDate, tags)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Todo), args.Error(1)
}

func (m *MockTodoService) DeleteTodo(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTodoService) DeleteCompletedTodos(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockTodoService) GetDeletedTodos(ctx context.Context) ([]model.Todo, error) {
	args := m.Called(ctx)
	return args.Get(0).([]model.Todo), args.Error(1)
}

func TestTodoHandler_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockTodoService)
		h := NewTodoHandler(mockSvc)

		mockSvc.On("CreateTodo", mock.Anything, "Test Todo", (*time.Time)(nil), ([]string)(nil)).
			Return(&model.Todo{ID: 1, Title: "Test Todo", Completed: false}, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := `{"title": "Test Todo"}`
		c.Request = httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBufferString(reqBody))
		c.Request.Header.Set("Content-Type", "application/json")

		h.Create(c)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp model.Todo
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Test Todo", resp.Title)
		assert.Equal(t, uint(1), resp.ID)
		mockSvc.AssertExpectations(t)
	})

	t.Run("bad request - missing title", func(t *testing.T) {
		mockSvc := new(MockTodoService)
		h := NewTodoHandler(mockSvc)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		reqBody := `{}`
		c.Request = httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBufferString(reqBody))
		c.Request.Header.Set("Content-Type", "application/json")

		h.Create(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestTodoHandler_Get(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockTodoService)
		h := NewTodoHandler(mockSvc)

		mockSvc.On("GetTodo", mock.Anything, uint(1)).
			Return(&model.Todo{ID: 1, Title: "Test Todo", Completed: false}, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = []gin.Param{{Key: "id", Value: "1"}}
		c.Request = httptest.NewRequest(http.MethodGet, "/todos/1", nil)

		h.Get(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp model.Todo
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, uint(1), resp.ID)
		mockSvc.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockSvc := new(MockTodoService)
		h := NewTodoHandler(mockSvc)

		mockSvc.On("GetTodo", mock.Anything, uint(999)).
			Return(nil, repository.ErrNotFound)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = []gin.Param{{Key: "id", Value: "999"}}
		c.Request = httptest.NewRequest(http.MethodGet, "/todos/999", nil)

		h.Get(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockSvc.AssertExpectations(t)
	})
}

func TestTodoHandler_GetAll(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockTodoService)
		h := NewTodoHandler(mockSvc)

		mockSvc.On("GetAllTodos", mock.Anything, model.TodoFilter{}).
			Return([]model.Todo{{ID: 1, Title: "Todo 1"}, {ID: 2, Title: "Todo 2"}}, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/todos", nil)

		h.GetAll(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp []model.Todo
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Len(t, resp, 2)
		mockSvc.AssertExpectations(t)
	})
}

func TestTodoHandler_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockTodoService)
		h := NewTodoHandler(mockSvc)

		mockSvc.On("UpdateTodo", mock.Anything, uint(1), "Updated Todo", true, (*time.Time)(nil), ([]string)(nil)).
			Return(&model.Todo{ID: 1, Title: "Updated Todo", Completed: true}, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = []gin.Param{{Key: "id", Value: "1"}}

		reqBody := `{"title": "Updated Todo", "completed": true}`
		c.Request = httptest.NewRequest(http.MethodPut, "/todos/1", bytes.NewBufferString(reqBody))
		c.Request.Header.Set("Content-Type", "application/json")

		h.Update(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp model.Todo
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "Updated Todo", resp.Title)
		assert.True(t, resp.Completed)
		mockSvc.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockSvc := new(MockTodoService)
		h := NewTodoHandler(mockSvc)

		mockSvc.On("UpdateTodo", mock.Anything, uint(999), "Title", false, (*time.Time)(nil), ([]string)(nil)).
			Return(nil, repository.ErrNotFound)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = []gin.Param{{Key: "id", Value: "999"}}

		reqBody := `{"title": "Title"}`
		c.Request = httptest.NewRequest(http.MethodPut, "/todos/999", bytes.NewBufferString(reqBody))
		c.Request.Header.Set("Content-Type", "application/json")

		h.Update(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockSvc.AssertExpectations(t)
	})
}

func TestTodoHandler_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockTodoService)
		h := NewTodoHandler(mockSvc)

		mockSvc.On("DeleteTodo", mock.Anything, uint(1)).Return(nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = []gin.Param{{Key: "id", Value: "1"}}
		c.Request = httptest.NewRequest(http.MethodDelete, "/todos/1", nil)

		h.Delete(c)

		assert.Equal(t, http.StatusNoContent, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockSvc := new(MockTodoService)
		h := NewTodoHandler(mockSvc)

		mockSvc.On("DeleteTodo", mock.Anything, uint(999)).Return(repository.ErrNotFound)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = []gin.Param{{Key: "id", Value: "999"}}
		c.Request = httptest.NewRequest(http.MethodDelete, "/todos/999", nil)

		h.Delete(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockSvc.AssertExpectations(t)
	})
}

func TestTodoHandler_DeleteCompleted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockTodoService)
		h := NewTodoHandler(mockSvc)

		mockSvc.On("DeleteCompletedTodos", mock.Anything).Return(nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/todos/clear-completed", nil)

		h.DeleteCompleted(c)

		assert.Equal(t, http.StatusNoContent, w.Code)
		mockSvc.AssertExpectations(t)
	})
}

func TestTodoHandler_GetDeleted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockTodoService)
		h := NewTodoHandler(mockSvc)

		mockSvc.On("GetDeletedTodos", mock.Anything).
			Return([]model.Todo{{ID: 1, Title: "Deleted Todo 1"}}, nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/todos/deleted", nil)

		h.GetDeleted(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp []model.Todo
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Len(t, resp, 1)
		assert.Equal(t, "Deleted Todo 1", resp[0].Title)
		mockSvc.AssertExpectations(t)
	})
}
