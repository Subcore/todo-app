package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Проверяем что GET /healthz всегда возвращает 200 {"status":"ok"}
func TestHealthHandler_Healthz(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	h := NewHealthHandler(nil)
	r.GET("/healthz", h.Healthz)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"status":"ok"}`, w.Body.String())
}

// Проверяем что GET /readyz возвращает 503 когда база не настроена
func TestHealthHandler_Readyz_Fail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	// Передаём nil вместо базы — это активирует путь "недоступно" без паники
	h := NewHealthHandler(nil)
	r.GET("/readyz", h.Readyz)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/readyz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.JSONEq(t, `{"status":"unavailable"}`, w.Body.String())
}
