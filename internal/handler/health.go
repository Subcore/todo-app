package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthHandler обрабатывает запросы проверки состояния сервиса
type HealthHandler struct {
	db *gorm.DB
}

// NewHealthHandler создаёт новый обработчик проверки состояния с подключением к БД
func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// Healthz godoc
// @Summary Проверка жизнеспособности
// @Description Возвращает 200 OK если сервис запущен
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /healthz [get]
func (h *HealthHandler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readyz godoc
// @Summary Проверка готовности
// @Description Возвращает 200 OK если сервис готов принимать трафик (соединение с БД активно)
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /readyz [get]
func (h *HealthHandler) Readyz(c *gin.Context) {
	if h.db == nil {
		log.Printf("[Readyz] Database instance is nil")
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
		return
	}

	db, err := h.db.DB()
	if err == nil {
		err = db.Ping()
	}

	if err != nil {
		log.Printf("[Readyz] Database health check failed: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
