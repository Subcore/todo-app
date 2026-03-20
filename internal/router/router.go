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

	// Static assets
	r.Static("/assets", "./web/assets")

	// Frontend
	r.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})

	// OpenAPI 3.0 spec
	r.StaticFile("/openapi.yaml", "./openapi.yaml")

	// Swagger UI
	r.GET("/docs", func(c *gin.Context) {
		c.File("./web/docs.html")
	})
	r.GET("/docs/", func(c *gin.Context) {
		c.Redirect(301, "/docs")
	})

	// Initialize layers
	repo := repository.NewTodoRepository(db)
	svc := service.NewTodoService(repo)
	h := handler.NewTodoHandler(svc)
	hh := handler.NewHealthHandler(db)

	// Health endpoints
	r.GET("/healthz", hh.Healthz)
	r.GET("/readyz", hh.Readyz)

	// API v1
	api := r.Group("/api/v1")
	{
		todoGroup := api.Group("/todos")
		{
			todoGroup.POST("", h.Create)
			todoGroup.GET("", h.GetAll)
			todoGroup.GET("/:id", h.Get)
			todoGroup.PUT("/:id", h.Update)
			todoGroup.DELETE("/:id", h.Delete)
			todoGroup.POST("/clear-completed", h.DeleteCompleted)
			todoGroup.GET("/deleted", h.GetDeleted)
		}
	}

	return r
}
