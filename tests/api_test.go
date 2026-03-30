package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"testing"

	"github.com/Subcore/todo-app-v2/internal/router"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// Проверяем весь HTTP-стек от роутера до ответа — без базы данных
func TestAPI_Healthz(t *testing.T) {
	r := router.SetupRouter(nil)
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
}

// Проверяем что /readyz возвращает 503 когда база не настроена
func TestAPI_Readyz_NoDB(t *testing.T) {
	r := router.SetupRouter(nil)
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/readyz")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

// Проверяем что /readyz возвращает 200 {"status":"ok"} когда база доступна — важно для Kubernetes-проб готовности
func TestAPI_Readyz_WithDB(t *testing.T) {
	db := setupTestDB(t)

	r := router.SetupRouter(db)
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/readyz")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
}

// Проверяем полный цикл создание → список → получение → удаление через реальный HTTP
func TestAPI_CRUD_Todo(t *testing.T) {
	db := setupTestDB(t)

	r := router.SetupRouter(db)
	srv := httptest.NewServer(r)
	defer srv.Close()

	client := srv.Client()
	base := srv.URL + "/api/v1/todos"

	// --- POST /api/v1/todos → 201 ---
	payload := `{"title":"integration test todo"}`
	postResp, err := client.Post(base, "application/json", bytes.NewBufferString(payload))
	require.NoError(t, err)
	defer postResp.Body.Close()

	assert.Equal(t, http.StatusCreated, postResp.StatusCode)

	var created map[string]interface{}
	require.NoError(t, json.NewDecoder(postResp.Body).Decode(&created))
	assert.Equal(t, "integration test todo", created["title"])

	id := created["id"]
	require.NotNil(t, id, "response must contain id")

	// --- GET /api/v1/todos → 200 + todo is visible ---
	getResp, err := client.Get(base)
	require.NoError(t, err)
	defer getResp.Body.Close()

	assert.Equal(t, http.StatusOK, getResp.StatusCode)

	var todos []map[string]interface{}
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&todos))

	found := false
	for _, todo := range todos {
		if todo["title"] == "integration test todo" {
			found = true
			break
		}
	}
	assert.True(t, found, "created todo must appear in GET /api/v1/todos")

	// --- GET /api/v1/todos/:id → 200 ---
	// JSON декодирует числа как float64, поэтому конвертируем перед сборкой URL
	idFloat, ok := id.(float64)
	require.True(t, ok, "id must be a number")
	idURL := fmt.Sprintf("%s/%.0f", base, idFloat)

	getOneResp, err := client.Get(idURL)
	require.NoError(t, err)
	defer getOneResp.Body.Close()

	assert.Equal(t, http.StatusOK, getOneResp.StatusCode)

	var fetched map[string]interface{}
	require.NoError(t, json.NewDecoder(getOneResp.Body).Decode(&fetched))
	assert.Equal(t, "integration test todo", fetched["title"])

	// --- DELETE /api/v1/todos/:id → 204 ---
	delReq, err := http.NewRequest(http.MethodDelete, idURL, nil)
	require.NoError(t, err)

	delResp, err := client.Do(delReq)
	require.NoError(t, err)
	defer delResp.Body.Close()

	assert.Equal(t, http.StatusNoContent, delResp.StatusCode)

	// --- DELETE same ID again → 404 (already soft-deleted) ---
	delReq2, err := http.NewRequest(http.MethodDelete, idURL, nil)
	require.NoError(t, err)

	delResp2, err := client.Do(delReq2)
	require.NoError(t, err)
	defer delResp2.Body.Close()

	assert.Equal(t, http.StatusNotFound, delResp2.StatusCode)
}

