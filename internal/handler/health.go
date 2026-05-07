package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthHandler serves the service health-check endpoints.
type HealthHandler struct {
	db *gorm.DB
}

// NewHealthHandler creates a new health handler bound to the given DB handle.
func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// Healthz godoc
// @Summary Liveness check
// @Description Returns 200 OK as long as the service is running.
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /healthz [get]
func (h *HealthHandler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readyz godoc
// @Summary Readiness check
// @Description Returns 200 OK once the service is ready to accept traffic (database connection is healthy).
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
