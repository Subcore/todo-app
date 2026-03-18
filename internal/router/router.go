package router

import (
	"github.com/Subcore/todo-app-v2/internal/handler"
	"github.com/Subcore/todo-app-v2/internal/repository"
	"github.com/Subcore/todo-app-v2/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()

	// Initialize repository
	repo := repository.NewTodoRepository(db)
	// Initialize service
	svc := service.NewTodoService(repo)
	// Initialize handler
	h := handler.NewTodoHandler(svc)

	api := r.Group("/api/v1")
	{
		todoGroup := api.Group("/todos")
		{
			todoGroup.POST("", h.Create)
			todoGroup.GET("", h.GetAll)
			todoGroup.GET("/:id", h.Get)
			todoGroup.PUT("/:id", h.Update)
			todoGroup.DELETE("/:id", h.Delete)
		}
	}

	return r
}

