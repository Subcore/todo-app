package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

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

// Note: Testing Readyz with a real DB connection failure is tricky without mocks.
// For now, I'll just verify the 200 case if I can, and skip the 503 if no mock is available.
// But wait, I can use a mock or a nil DB to see how it behaves.
func TestHealthHandler_Readyz_Fail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	
	// With the nil check added, it should not panic but return 503
	h := NewHealthHandler(nil) 
	r.GET("/readyz", h.Readyz)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/readyz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.JSONEq(t, `{"status":"unavailable"}`, w.Body.String())
}
