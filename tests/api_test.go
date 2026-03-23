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

// TestAPI_Healthz verifies the full HTTP stack: router → handler → response.
// Runs without a database.
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

// TestAPI_Readyz_NoDB verifies the readiness endpoint returns 503 when no DB is configured.
func TestAPI_Readyz_NoDB(t *testing.T) {
	r := router.SetupRouter(nil)
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/readyz")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}



// TestAPI_CRUD_Todo tests the full create → list → verify lifecycle over real HTTP.
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

// TestAPI_CreateTodo_EmptyTitle verifies that creating a todo with empty title returns 400.
func TestAPI_CreateTodo_EmptyTitle(t *testing.T) {
	db := setupTestDB(t)

	r := router.SetupRouter(db)
	srv := httptest.NewServer(r)
	defer srv.Close()

	// Empty string title
	resp, err := srv.Client().Post(
		srv.URL+"/api/v1/todos",
		"application/json",
		bytes.NewBufferString(`{"title":""}`),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestAPI_CreateTodo_MissingBody verifies that sending no body returns 400.
func TestAPI_CreateTodo_MissingBody(t *testing.T) {
	db := setupTestDB(t)

	r := router.SetupRouter(db)
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := srv.Client().Post(
		srv.URL+"/api/v1/todos",
		"application/json",
		bytes.NewBufferString(`{}`),
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestAPI_GetTodo_NotFound verifies that fetching a non-existent todo returns 404.
func TestAPI_GetTodo_NotFound(t *testing.T) {
	db := setupTestDB(t)

	r := router.SetupRouter(db)
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/api/v1/todos/999999")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
