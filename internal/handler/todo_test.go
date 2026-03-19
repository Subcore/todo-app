package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Subcore/todo-app-v2/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// mockTodoService implements service.TodoService interface for testing
type mockTodoService struct {
	CreateTodoFunc func(ctx context.Context, title string) (*model.Todo, error)
	GetTodoFunc    func(ctx context.Context, id uint) (*model.Todo, error)
	GetAllTodosFunc func(ctx context.Context) ([]model.Todo, error)
	UpdateTodoFunc func(ctx context.Context, id uint, title string, completed bool) (*model.Todo, error)
	DeleteTodoFunc func(ctx context.Context, id uint) error
	DeleteCompletedTodosFunc func(ctx context.Context) error
	GetDeletedTodosFunc    func(ctx context.Context) ([]model.Todo, error)
}

func (m *mockTodoService) CreateTodo(ctx context.Context, title string) (*model.Todo, error) {
	return m.CreateTodoFunc(ctx, title)
}

func (m *mockTodoService) GetTodo(ctx context.Context, id uint) (*model.Todo, error) {
	return m.GetTodoFunc(ctx, id)
}

func (m *mockTodoService) GetAllTodos(ctx context.Context) ([]model.Todo, error) {
	return m.GetAllTodosFunc(ctx)
}

func (m *mockTodoService) UpdateTodo(ctx context.Context, id uint, title string, completed bool) (*model.Todo, error) {
	return m.UpdateTodoFunc(ctx, id, title, completed)
}

func (m *mockTodoService) DeleteTodo(ctx context.Context, id uint) error {
	return m.DeleteTodoFunc(ctx, id)
}

func (m *mockTodoService) DeleteCompletedTodos(ctx context.Context) error {
	return m.DeleteCompletedTodosFunc(ctx)
}

func (m *mockTodoService) GetDeletedTodos(ctx context.Context) ([]model.Todo, error) {
	return m.GetDeletedTodosFunc(ctx)
}

func TestTodoHandler_Create(t *testing.T) {
	// Set Gin to Test Mode
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		// Mock service
		mockSvc := &mockTodoService{
			CreateTodoFunc: func(ctx context.Context, title string) (*model.Todo, error) {
				return &model.Todo{ID: 1, Title: title, Completed: false}, nil
			},
		}
		h := NewTodoHandler(mockSvc)

		// Create response recorder
		w := httptest.NewRecorder()
		// Create gin context
		c, _ := gin.CreateTestContext(w)

		// Create fake request
		reqBody := `{"title": "Test Todo"}`
		c.Request = httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBufferString(reqBody))
		c.Request.Header.Set("Content-Type", "application/json")

		// Call handler
		h.Create(c)

		// Assertions
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp model.Todo
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "Test Todo", resp.Title)
		assert.Equal(t, uint(1), resp.ID)
	})

	t.Run("bad request - missing title", func(t *testing.T) {
		mockSvc := &mockTodoService{}
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
		mockSvc := &mockTodoService{
			GetTodoFunc: func(ctx context.Context, id uint) (*model.Todo, error) {
				return &model.Todo{ID: id, Title: "Test Todo", Completed: false}, nil
			},
		}
		h := NewTodoHandler(mockSvc)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = []gin.Param{{Key: "id", Value: "1"}}

		c.Request = httptest.NewRequest(http.MethodGet, "/todos/1", nil)

		h.Get(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp model.Todo
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, uint(1), resp.ID)
	})

	t.Run("not found", func(t *testing.T) {
		mockSvc := &mockTodoService{
			GetTodoFunc: func(ctx context.Context, id uint) (*model.Todo, error) {
				return nil, context.DeadlineExceeded
			},
		}
		h := NewTodoHandler(mockSvc)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = []gin.Param{{Key: "id", Value: "999"}}

		c.Request = httptest.NewRequest(http.MethodGet, "/todos/999", nil)

		h.Get(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestTodoHandler_GetAll(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mockSvc := &mockTodoService{
			GetAllTodosFunc: func(ctx context.Context) ([]model.Todo, error) {
				return []model.Todo{{ID: 1, Title: "Todo 1"}, {ID: 2, Title: "Todo 2"}}, nil
			},
		}
		h := NewTodoHandler(mockSvc)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/todos", nil)

		h.GetAll(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp []model.Todo
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Len(t, resp, 2)
	})
}

func TestTodoHandler_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mockSvc := &mockTodoService{
			UpdateTodoFunc: func(ctx context.Context, id uint, title string, completed bool) (*model.Todo, error) {
				return &model.Todo{ID: id, Title: title, Completed: completed}, nil
			},
		}
		h := NewTodoHandler(mockSvc)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = []gin.Param{{Key: "id", Value: "1"}}

		reqBody := `{"title": "Updated Todo", "completed": true}`
		c.Request = httptest.NewRequest(http.MethodPut, "/todos/1", bytes.NewBufferString(reqBody))
		c.Request.Header.Set("Content-Type", "application/json")

		h.Update(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp model.Todo
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "Updated Todo", resp.Title)
		assert.True(t, resp.Completed)
	})
}

func TestTodoHandler_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mockSvc := &mockTodoService{
			DeleteTodoFunc: func(ctx context.Context, id uint) error {
				return nil
			},
		}
		h := NewTodoHandler(mockSvc)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = []gin.Param{{Key: "id", Value: "1"}}

		c.Request = httptest.NewRequest(http.MethodDelete, "/todos/1", nil)

		h.Delete(c)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

func TestTodoHandler_DeleteCompleted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mockSvc := &mockTodoService{
			DeleteCompletedTodosFunc: func(ctx context.Context) error {
				return nil
			},
		}
		h := NewTodoHandler(mockSvc)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodPost, "/todos/clear-completed", nil)

		h.DeleteCompleted(c)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

func TestTodoHandler_GetDeleted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("success", func(t *testing.T) {
		mockSvc := &mockTodoService{
			GetDeletedTodosFunc: func(ctx context.Context) ([]model.Todo, error) {
				return []model.Todo{{ID: 1, Title: "Deleted Todo 1"}}, nil
			},
		}
		h := NewTodoHandler(mockSvc)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		c.Request = httptest.NewRequest(http.MethodGet, "/todos/deleted", nil)

		h.GetDeleted(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp []model.Todo
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Len(t, resp, 1)
		assert.Equal(t, "Deleted Todo 1", resp[0].Title)
	})
}
