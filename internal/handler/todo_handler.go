package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Subcore/todo-app-v2/internal/model"
	"github.com/Subcore/todo-app-v2/internal/service"
	"github.com/gin-gonic/gin"
)

type TodoHandler struct {
	svc service.TodoService
}

func NewTodoHandler(svc service.TodoService) *TodoHandler {
	return &TodoHandler{svc: svc}
}

type createTodoRequest struct {
	Title   string     `json:"title" binding:"required"`
	DueDate *time.Time `json:"due_date"`
	Tags    []string   `json:"tags"`
}

type updateTodoRequest struct {
	Title     string     `json:"title" binding:"required"`
	Completed bool       `json:"completed"`
	DueDate   *time.Time `json:"due_date"`
	Tags      []string   `json:"tags"`
}

type getTodosQuery struct {
	Completed *bool      `form:"completed"`
	Search    string     `form:"search"`
	DueBefore *time.Time `form:"due_before" time_format:"2006-01-02T15:04:05Z07:00"`
	DueAfter  *time.Time `form:"due_after" time_format:"2006-01-02T15:04:05Z07:00"`
}

// Create godoc
// @Summary Create a todo
// @Description Create a new todo item with a title
// @Tags todos
// @Accept json
// @Produce json
// @Param request body createTodoRequest true "Todo request"
// @Success 201 {object} model.Todo
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /todos [post]
func (h *TodoHandler) Create(c *gin.Context) {
	var req createTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	todo, err := h.svc.CreateTodo(c.Request.Context(), req.Title, req.DueDate, req.Tags)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, todo)
}

// Get godoc
// @Summary Get a todo by ID
// @Description Retrieve a todo item using its ID
// @Tags todos
// @Accept json
// @Produce json
// @Param id path uint true "Todo ID"
// @Success 200 {object} model.Todo
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /todos/{id} [get]
func (h *TodoHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	todo, err := h.svc.GetTodo(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.JSON(http.StatusOK, todo)
}

// GetAll godoc
// @Summary Get all todos
// @Description Retrieve a list of all todo items
// @Tags todos
// @Accept json
// @Produce json
// @Success 200 {array} model.Todo
// @Failure 500 {object} map[string]string
// @Router /todos [get]
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
// @Description Update title or status of a todo item
// @Tags todos
// @Accept json
// @Produce json
// @Param id path uint true "Todo ID"
// @Param request body updateTodoRequest true "Todo update request"
// @Success 200 {object} model.Todo
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /todos/{id} [put]
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, todo)
}

// Delete godoc
// @Summary Delete a todo (Soft Delete)
// @Description Delete a todo item by its ID. This is a soft delete (using GORM's gorm.DeletedAt). The record remains in the database with a deleted_at timestamp.
// @Tags todos
// @Accept json
// @Produce json
// @Param id path uint true "Todo ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /todos/{id} [delete]
func (h *TodoHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	if err := h.svc.DeleteTodo(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// DeleteCompleted godoc
// @Summary Delete all completed todos (Soft Delete)
// @Description Delete all todo items that are marked as completed. This is a soft delete (using GORM's gorm.DeletedAt).
// @Tags todos
// @Accept json
// @Produce json
// @Success 204 "No Content"
// @Failure 500 {object} map[string]string
// @Router /todos/clear-completed [post]
func (h *TodoHandler) DeleteCompleted(c *gin.Context) {
	if err := h.svc.DeleteCompletedTodos(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// GetDeleted godoc
// @Summary Get all deleted todos
// @Description Retrieve a list of all soft-deleted todo items
// @Tags todos
// @Accept json
// @Produce json
// @Success 200 {array} model.Todo
// @Failure 500 {object} map[string]string
// @Router /todos/deleted [get]
func (h *TodoHandler) GetDeleted(c *gin.Context) {
	todos, err := h.svc.GetDeletedTodos(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, todos)
}
