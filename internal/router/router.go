package router

import (
	"github.com/Subcore/todo-app-v2/internal/handler"
	"github.com/Subcore/todo-app-v2/internal/repository"
	"github.com/Subcore/todo-app-v2/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRouter настраивает маршрутизатор с подключением к базе данных
func SetupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()

	// Статические ресурсы
	r.Static("/assets", "./web/assets")

	// Главная страница фронтенда
	r.GET("/", func(c *gin.Context) {
		c.File("./web/index.html")
	})

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

	// Эндпоинты проверки состояния
	r.GET("/healthz", hh.Healthz)
	r.GET("/readyz", hh.Readyz)

	// API версии 1
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
