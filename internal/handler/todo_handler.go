package handler

import (
	"net/http"
	"strconv"

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
	Title string `json:"title" binding:"required"`
}

type updateTodoRequest struct {
	Title     string `json:"title" binding:"required"`
	Completed bool   `json:"completed"`
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

	todo, err := h.svc.CreateTodo(c.Request.Context(), req.Title)
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
	todos, err := h.svc.GetAllTodos(c.Request.Context())
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

	todo, err := h.svc.UpdateTodo(c.Request.Context(), uint(id), req.Title, req.Completed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, todo)
}

// Delete godoc
// @Summary Delete a todo
// @Description Delete a todo item by its ID
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
