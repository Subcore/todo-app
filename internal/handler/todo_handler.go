package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Subcore/todo-app-v2/internal/model"
	"github.com/Subcore/todo-app-v2/internal/repository"
	"github.com/Subcore/todo-app-v2/internal/service"
	"github.com/gin-gonic/gin"
)

// TodoHandler обрабатывает HTTP-запросы для работы с задачами
type TodoHandler struct {
	svc service.TodoService
}

// NewTodoHandler создаёт новый обработчик задач с переданным сервисом
func NewTodoHandler(svc service.TodoService) *TodoHandler {
	return &TodoHandler{svc: svc}
}

// createTodoRequest — тело запроса на создание задачи
type createTodoRequest struct {
	Title   string     `json:"title" binding:"required,max=255"`
	DueDate *time.Time `json:"due_date"`
	Tags    []string   `json:"tags"`
}

// updateTodoRequest — тело запроса на обновление задачи
type updateTodoRequest struct {
	Title     string     `json:"title" binding:"required,max=255"`
	Completed *bool      `json:"completed"`
	DueDate   *time.Time `json:"due_date"`
	Tags      []string   `json:"tags"`
}

// getTodosQuery — параметры запроса для фильтрации списка задач
type getTodosQuery struct {
	Completed *bool      `form:"completed"`
	Search    string     `form:"search"`
	DueBefore *time.Time `form:"due_before" time_format:"2006-01-02T15:04:05Z07:00"`
	DueAfter  *time.Time `form:"due_after" time_format:"2006-01-02T15:04:05Z07:00"`
}

// Create godoc
// @Summary Создать задачу
// @Description Создаёт новую задачу с заголовком
// @Tags todos
// @Accept json
// @Produce json
// @Param request body createTodoRequest true "Тело запроса задачи"
// @Success 201 {object} model.Todo
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/todos [post]
func (h *TodoHandler) Create(c *gin.Context) {
	var req createTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	todo, err := h.svc.CreateTodo(c.Request.Context(), req.Title, req.DueDate, req.Tags)
	if err != nil {
		if errors.Is(err, service.ErrEmptyTitle) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, todo)
}

// Get godoc
// @Summary Получить задачу по ID
// @Description Возвращает задачу по её идентификатору
// @Tags todos
// @Accept json
// @Produce json
// @Param id path uint true "ID задачи"
// @Success 200 {object} model.Todo
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/todos/{id} [get]
func (h *TodoHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	todo, err := h.svc.GetTodo(c.Request.Context(), uint(id))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, todo)
}

// GetAll godoc
// @Summary Получить все задачи
// @Description Возвращает список всех задач с возможностью фильтрации
// @Tags todos
// @Accept json
// @Produce json
// @Param completed   query bool   false "Фильтр по статусу выполнения"
// @Param search      query string false "Поиск по заголовку (без учёта регистра)"
// @Param due_before  query string false "Срок до (RFC3339)"
// @Param due_after   query string false "Срок после (RFC3339)"
// @Success 200 {array} model.Todo
// @Failure 500 {object} map[string]string
// @Router /api/v1/todos [get]
func (h *TodoHandler) GetAll(c *gin.Context) {
	var query getTodosQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter := model.TodoFilter{
		Completed: query.Completed,
		Search:    query.Search,
		DueBefore: query.DueBefore,
		DueAfter:  query.DueAfter,
	}

	todos, err := h.svc.GetAllTodos(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, todos)
}

// Update godoc
// @Summary Обновить задачу
// @Description Обновляет заголовок или статус задачи
// @Tags todos
// @Accept json
// @Produce json
// @Param id path uint true "ID задачи"
// @Param request body updateTodoRequest true "Тело запроса обновления задачи"
// @Success 200 {object} model.Todo
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/todos/{id} [put]
func (h *TodoHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var req updateTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	todo, err := h.svc.UpdateTodo(c.Request.Context(), uint(id), req.Title, req.Completed, req.DueDate, req.Tags)
	if err != nil {
		if errors.Is(err, service.ErrEmptyTitle) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, todo)
}

// Delete godoc
// @Summary Удалить задачу (мягкое удаление)
// @Description Удаляет задачу по ID. Это мягкое удаление (через gorm.DeletedAt). Запись остаётся в базе с меткой времени deleted_at.
// @Tags todos
// @Accept json
// @Produce json
// @Param id path uint true "ID задачи"
// @Success 204 "Нет содержимого"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/todos/{id} [delete]
func (h *TodoHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	if err := h.svc.DeleteTodo(c.Request.Context(), uint(id)); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
	c.Writer.WriteHeaderNow()
}

// DeleteCompleted godoc
// @Summary Удалить все выполненные задачи (мягкое удаление)
// @Description Удаляет все задачи со статусом «выполнено». Это мягкое удаление (через gorm.DeletedAt).
// @Tags todos
// @Accept json
// @Produce json
// @Success 204 "Нет содержимого"
// @Failure 500 {object} map[string]string
// @Router /api/v1/todos/clear-completed [post]
func (h *TodoHandler) DeleteCompleted(c *gin.Context) {
	if err := h.svc.DeleteCompletedTodos(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
	c.Writer.WriteHeaderNow()
}

// GetDeleted godoc
// @Summary Получить все удалённые задачи
// @Description Возвращает список всех мягко удалённых задач
// @Tags todos
// @Accept json
// @Produce json
// @Success 200 {array} model.Todo
// @Failure 500 {object} map[string]string
// @Router /api/v1/todos/deleted [get]
func (h *TodoHandler) GetDeleted(c *gin.Context) {
	todos, err := h.svc.GetDeletedTodos(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, todos)
}
