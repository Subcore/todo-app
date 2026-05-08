package router

import (
	"os"

	"github.com/Subcore/todo-app/internal/handler"
	"github.com/Subcore/todo-app/internal/repository"
	"github.com/Subcore/todo-app/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRouter wires up the HTTP router with the given DB handle.
func SetupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://todo.local", "http://localhost", "http://localhost:80"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
	}))

	repo := repository.NewTodoRepository(db)
	svc := service.NewTodoService(repo)
	h := handler.NewTodoHandler(svc)
	hh := handler.NewHealthHandler(db)

	// Health endpoints are served under two paths for flexibility.
	r.GET("/healthz", hh.Healthz)
	r.GET("/readyz", hh.Readyz)

	api := r.Group("/api")
	{
		// Swagger UI — only mounted when ENABLE_SWAGGER=true.
		if os.Getenv("ENABLE_SWAGGER") == "true" {
			api.StaticFile("/openapi.yaml", "./openapi.yaml")
			api.GET("/docs", func(c *gin.Context) {
				c.File("./web/docs.html")
			})
			api.GET("/docs/", func(c *gin.Context) {
				c.Redirect(301, "/api/docs")
			})
		}

		api.GET("/healthz", hh.Healthz)
		api.GET("/readyz", hh.Readyz)

		v1 := api.Group("/v1")
		{
			v1.GET("/healthz", hh.Healthz)
			v1.GET("/readyz", hh.Readyz)

			todoGroup := v1.Group("/todos")
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
	}

	return r
}
