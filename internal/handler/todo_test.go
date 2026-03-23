package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Subcore/todo-app-v2/internal/model"
	"github.com/Subcore/todo-app-v2/internal/repository"
	"github.com/Subcore/todo-app-v2/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Проверка на этапе компиляции: MockTodoService реализует service.TodoService.
// Если интерфейс изменится — код не скомпилируется.
var _ service.TodoService = (*MockTodoService)(nil)

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

func (m *MockTodoService) UpdateTodo(ctx context.Context, id uint, title string, completed *bool, dueDate *time.Time, tags []string) (*model.Todo, error) {
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

// Проверяем что POST /todos создаёт задачу и возвращает 201, а при отсутствии заголовка — 400
func TestTodoHandler_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Нормальный заголовок — задача создаётся, ответ 201 с данными задачи
	t.Run("success", func(t *testing.T) {
		// Подставляем фейковый сервис вместо настоящего
		mockSvc := new(MockTodoService)
		h := NewTodoHandler(mockSvc)

		// Говорим фейку: когда вызовут CreateTodo с "Test Todo" — вернуть объект задачи
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
		// Проверяем что фейк вызвали именно так, как договаривались
		mockSvc.AssertExpectations(t)
	})

	// Тело запроса без заголовка должно отклоняться до вызова сервиса
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
		// Проверяем что до сервиса дело не дошло
		mockSvc.AssertNotCalled(t, "CreateTodo")
	})
}

// Проверяем что GET /todos/:id возвращает задачу при успехе и 404 если не найдена
func TestTodoHandler_Get(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Существующий ID — ответ 200 с данными задачи
	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockTodoService)
		h := NewTodoHandler(mockSvc)

		// Говорим фейку: вернуть задачу при запросе ID=1
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

	// Запрос несуществующей задачи — ответ 404
	t.Run("not found", func(t *testing.T) {
		mockSvc := new(MockTodoService)
		h := NewTodoHandler(mockSvc)

		// Говорим фейку: задача 999 не существует
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

// Проверяем что GET /todos возвращает полный список задач
func TestTodoHandler_GetAll(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// В базе две задачи — обе должны попасть в ответ
	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockTodoService)
		h := NewTodoHandler(mockSvc)

		// Говорим фейку: вернуть две задачи при пустом фильтре
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

// Проверяем что PUT /todos/:id обновляет задачу и возвращает 200, или 404 если не найдена
func TestTodoHandler_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Заголовок и статус обновляются — ответ отражает изменения
	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockTodoService)
		h := NewTodoHandler(mockSvc)

		trueVal := true
		// Говорим фейку: принять обновление ID=1 с новым заголовком и completed=true
		mockSvc.On("UpdateTodo", mock.Anything, uint(1), "Updated Todo", &trueVal, (*time.Time)(nil), ([]string)(nil)).
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

	// Обновление несуществующей задачи — ответ 404
	t.Run("not found", func(t *testing.T) {
		mockSvc := new(MockTodoService)
		h := NewTodoHandler(mockSvc)

		// Говорим фейку: задача 999 не существует
		mockSvc.On("UpdateTodo", mock.Anything, uint(999), "Title", (*bool)(nil), (*time.Time)(nil), ([]string)(nil)).
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

// Проверяем что DELETE /todos/:id помечает задачу как удалённую и возвращает 204, или 404 если не найдена
func TestTodoHandler_Delete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Существующая задача удаляется — ответ 204 без тела
	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockTodoService)
		h := NewTodoHandler(mockSvc)

		// Говорим фейку: удаление задачи 1 проходит успешно
		mockSvc.On("DeleteTodo", mock.Anything, uint(1)).Return(nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = []gin.Param{{Key: "id", Value: "1"}}
		c.Request = httptest.NewRequest(http.MethodDelete, "/todos/1", nil)

		h.Delete(c)

		assert.Equal(t, http.StatusNoContent, w.Code)
		mockSvc.AssertExpectations(t)
	})

	// Удаление несуществующей задачи — ответ 404
	t.Run("not found", func(t *testing.T) {
		mockSvc := new(MockTodoService)
		h := NewTodoHandler(mockSvc)

		// Говорим фейку: задача 999 не существует
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

// Проверяем что POST /todos/clear-completed возвращает 204 при успехе
func TestTodoHandler_DeleteCompleted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Все выполненные задачи помечаются как удалённые — ответ 204
	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockTodoService)
		h := NewTodoHandler(mockSvc)

		// Говорим фейку: удаление выполненных задач проходит успешно
		mockSvc.On("DeleteCompletedTodos", mock.Anything).Return(nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/todos/clear-completed", nil)

		h.DeleteCompleted(c)

		assert.Equal(t, http.StatusNoContent, w.Code)
		mockSvc.AssertExpectations(t)
	})
}

// boolPtr нужен чтобы получить указатель на булев литерал — в Go нельзя взять адрес у литерала напрямую
func boolPtr(b bool) *bool { return &b }

// Проверяем что нечисловой ID в URL возвращает 400 до вызова сервиса
func TestTodoHandler_Get_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockTodoService)
	h := NewTodoHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = []gin.Param{{Key: "id", Value: "abc"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/todos/abc", nil)

	h.Get(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "invalid ID", resp["error"])
	// Проверяем что до сервиса дело не дошло
	mockSvc.AssertNotCalled(t, "GetTodo")
}

// Проверяем что нечисловой ID при обновлении возвращает 400 до вызова сервиса
func TestTodoHandler_Update_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockTodoService)
	h := NewTodoHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = []gin.Param{{Key: "id", Value: "abc"}}

	reqBody := `{"title": "Updated"}`
	c.Request = httptest.NewRequest(http.MethodPut, "/todos/abc", bytes.NewBufferString(reqBody))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Update(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "invalid ID", resp["error"])
	// Проверяем что до сервиса дело не дошло
	mockSvc.AssertNotCalled(t, "UpdateTodo")
}

// Проверяем что нечисловой и отрицательный ID при удалении отклоняются с кодом 400
func TestTodoHandler_Delete_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Строка "abc" — не валидный ID
	t.Run("non-numeric id", func(t *testing.T) {
		mockSvc := new(MockTodoService)
		h := NewTodoHandler(mockSvc)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = []gin.Param{{Key: "id", Value: "abc"}}
		c.Request = httptest.NewRequest(http.MethodDelete, "/todos/abc", nil)

		h.Delete(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "invalid ID", resp["error"])
		// Проверяем что до сервиса дело не дошло
		mockSvc.AssertNotCalled(t, "DeleteTodo")
	})

	// Отрицательное число парсится, но не является допустимым ID сущности
	t.Run("negative id", func(t *testing.T) {
		mockSvc := new(MockTodoService)
		h := NewTodoHandler(mockSvc)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = []gin.Param{{Key: "id", Value: "-1"}}
		c.Request = httptest.NewRequest(http.MethodDelete, "/todos/-1", nil)

		h.Delete(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockSvc.AssertNotCalled(t, "DeleteTodo")
	})
}

// Проверяем что query-параметры разбираются в фильтр и передаются в сервис
func TestTodoHandler_GetAll_WithQueryParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := new(MockTodoService)
	h := NewTodoHandler(mockSvc)

	// Хэндлер должен собрать именно такой фильтр из строки запроса
	expectedFilter := model.TodoFilter{
		Completed: boolPtr(true),
		Search:    "test",
	}
	// Говорим фейку: вернуть одну подходящую задачу для заданного фильтра
	mockSvc.On("GetAllTodos", mock.Anything, expectedFilter).
		Return([]model.Todo{{ID: 1, Title: "test todo", Completed: true}}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/todos?completed=true&search=test", nil)

	h.GetAll(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []model.Todo
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp, 1)
	mockSvc.AssertExpectations(t)
}

// Проверяем что GET /todos/deleted возвращает список мягко удалённых задач
func TestTodoHandler_GetDeleted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Одна удалённая задача — должна вернуться в ответе
	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockTodoService)
		h := NewTodoHandler(mockSvc)

		// Говорим фейку: вернуть одну мягко удалённую задачу
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

// Проверяем что title длиной 256 символов отклоняется с кодом 400
func TestTodoHandler_Create_TitleTooLong(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockSvc := new(MockTodoService)
	h := NewTodoHandler(mockSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	longTitle := strings.Repeat("a", 256)
	reqBody := fmt.Sprintf(`{"title":%q}`, longTitle)
	c.Request = httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBufferString(reqBody))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockSvc.AssertNotCalled(t, "CreateTodo")
}

// Проверяем что title ровно 255 символов принимается
func TestTodoHandler_Create_TitleExactMax(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockSvc := new(MockTodoService)
	h := NewTodoHandler(mockSvc)

	exactTitle := strings.Repeat("a", 255)
	mockSvc.On("CreateTodo", mock.Anything, exactTitle, (*time.Time)(nil), ([]string)(nil)).
		Return(&model.Todo{ID: 1, Title: exactTitle}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	reqBody := fmt.Sprintf(`{"title":%q}`, exactTitle)
	c.Request = httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBufferString(reqBody))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Create(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	mockSvc.AssertExpectations(t)
}

// Проверяем что query-параметры due_before и due_after парсятся и передаются в сервис
func TestTodoHandler_GetAll_WithDateFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dueBefore := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	dueAfter := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		query  string
		filter model.TodoFilter
	}{
		{
			name:   "due_before only",
			query:  "/todos?due_before=" + dueBefore.Format(time.RFC3339),
			filter: model.TodoFilter{DueBefore: &dueBefore},
		},
		{
			name:   "due_after only",
			query:  "/todos?due_after=" + dueAfter.Format(time.RFC3339),
			filter: model.TodoFilter{DueAfter: &dueAfter},
		},
		{
			name:  "both dates",
			query: "/todos?due_before=" + dueBefore.Format(time.RFC3339) + "&due_after=" + dueAfter.Format(time.RFC3339),
			filter: model.TodoFilter{
				DueBefore: &dueBefore,
				DueAfter:  &dueAfter,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := new(MockTodoService)
			h := NewTodoHandler(mockSvc)

			mockSvc.On("GetAllTodos", mock.Anything, tt.filter).
				Return([]model.Todo{{ID: 1, Title: "Filtered"}}, nil)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, tt.query, nil)

			h.GetAll(c)

			assert.Equal(t, http.StatusOK, w.Code)
			mockSvc.AssertExpectations(t)
		})
	}
}
