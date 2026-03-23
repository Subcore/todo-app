package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gormPostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
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

// Проверяем что GET /readyz возвращает 200 когда база доступна и отвечает на ping
func TestHealthHandler_Readyz_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Создаём фейковое SQL-соединение, которое успешно отвечает на Ping
	sqlDB, mockSQL, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer sqlDB.Close()

	// Первый Ping — потребляется gorm.Open при инициализации соединения
	mockSQL.ExpectPing()
	// Второй Ping — вызывается в Readyz handler при проверке здоровья БД
	mockSQL.ExpectPing()

	// Оборачиваем в GORM
	gormDB, err := gorm.Open(gormPostgres.New(gormPostgres.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)

	r := gin.Default()
	h := NewHealthHandler(gormDB)
	r.GET("/readyz", h.Readyz)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/readyz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"status":"ok"}`, w.Body.String())
	require.NoError(t, mockSQL.ExpectationsWereMet())
}
