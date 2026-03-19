package router

import (
	_ "github.com/Subcore/todo-app-v2/docs"
	"github.com/Subcore/todo-app-v2/internal/handler"
	"github.com/Subcore/todo-app-v2/internal/repository"
	"github.com/Subcore/todo-app-v2/internal/service"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

func SetupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})

	// Initialize repository
	repo := repository.NewTodoRepository(db)
	// Initialize service
	svc := service.NewTodoService(repo)
	// Initialize handler
	h := handler.NewTodoHandler(svc)
	hh := handler.NewHealthHandler(db)

	r.GET("/healthz", hh.Healthz)
	r.GET("/readyz", hh.Readyz)

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

