package router

import (
	"github.com/Subcore/todo-app-v2/internal/handler"
	"github.com/Subcore/todo-app-v2/internal/repository"
	"github.com/Subcore/todo-app-v2/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRouter настраивает маршрутизатор с подключением к базе данных
func SetupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()

	// CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://todo.local", "http://localhost", "http://localhost:80"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
	}))

	// Спецификация OpenAPI 3.0
	r.StaticFile("/openapi.yaml", "./openapi.yaml")

	// Страница документации Swagger UI
	r.GET("/docs", func(c *gin.Context) {
		c.File("./web/docs.html")
	})
	r.GET("/docs/", func(c *gin.Context) {
		c.Redirect(301, "/docs")
	})

	// Инициализация слоёв приложения
	repo := repository.NewTodoRepository(db)
	svc := service.NewTodoService(repo)
	h := handler.NewTodoHandler(svc)
	hh := handler.NewHealthHandler(db)

	// Эндпоинты проверки состояния (обслуживаются на двух путях для гибкости)
	r.GET("/healthz", hh.Healthz)
	r.GET("/readyz", hh.Readyz)

	// API группа
	api := r.Group("/api")
	{
		// Эндпоинты здоровья также доступны через /api
		api.GET("/healthz", hh.Healthz)
		api.GET("/readyz", hh.Readyz)

		// API версии 1
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
