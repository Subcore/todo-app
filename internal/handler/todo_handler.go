package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Subcore/todo-app/internal/model"
	"github.com/Subcore/todo-app/internal/repository"
	"github.com/Subcore/todo-app/internal/service"
	"github.com/gin-gonic/gin"
)

// TodoHandler serves the HTTP endpoints for todo CRUD.
type TodoHandler struct {
	svc service.TodoService
}

// NewTodoHandler creates a new todo handler bound to the given service.
func NewTodoHandler(svc service.TodoService) *TodoHandler {
	return &TodoHandler{svc: svc}
}

// createTodoRequest is the request body for creating a todo.
type createTodoRequest struct {
	Title   string     `json:"title" binding:"required,max=255"`
	DueDate *time.Time `json:"due_date"`
	Tags    []string   `json:"tags"`
}

// updateTodoRequest is the request body for updating a todo.
type updateTodoRequest struct {
	Title     string     `json:"title" binding:"required,max=255"`
	Completed *bool      `json:"completed"`
	DueDate   *time.Time `json:"due_date"`
	Tags      []string   `json:"tags"`
}

// getTodosQuery groups the query parameters supported by the list endpoint.
type getTodosQuery struct {
	Completed *bool      `form:"completed"`
	Search    string     `form:"search"`
	DueBefore *time.Time `form:"due_before" time_format:"2006-01-02T15:04:05Z07:00"`
	DueAfter  *time.Time `form:"due_after" time_format:"2006-01-02T15:04:05Z07:00"`
}

// Create godoc
// @Summary Create a todo
// @Description Creates a new todo from the request body.
// @Tags todos
// @Accept json
// @Produce json
// @Param request body createTodoRequest true "Todo request body"
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
// @Summary Get a todo by ID
// @Description Returns a todo by its identifier.
// @Tags todos
// @Accept json
// @Produce json
// @Param id path uint true "Todo ID"
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
// @Summary List todos
// @Description Returns all todos, optionally filtered.
// @Tags todos
// @Accept json
// @Produce json
// @Param completed   query bool   false "Filter by completion status"
// @Param search      query string false "Case-insensitive title search"
// @Param due_before  query string false "Due before (RFC3339)"
// @Param due_after   query string false "Due after (RFC3339)"
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
// @Summary Update a todo
// @Description Updates a todo's title, completion status, due date, or tags.
// @Tags todos
// @Accept json
// @Produce json
// @Param id path uint true "Todo ID"
// @Param request body updateTodoRequest true "Todo update request body"
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
// @Summary Delete a todo (soft delete)
// @Description Deletes a todo by ID. This is a soft delete (via gorm.DeletedAt) — the row stays in the database with a deleted_at timestamp.
// @Tags todos
// @Accept json
// @Produce json
// @Param id path uint true "Todo ID"
// @Success 204 "No Content"
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
// @Summary Delete all completed todos (soft delete)
// @Description Deletes every todo whose completed flag is true. This is a soft delete (via gorm.DeletedAt).
// @Tags todos
// @Accept json
// @Produce json
// @Success 204 "No Content"
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
// @Summary List deleted todos
// @Description Returns every soft-deleted todo.
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
