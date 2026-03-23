package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

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

// TestAPI_Readyz_WithDB verifies that /readyz returns 200 {"status":"ok"} when the DB is reachable.
// Critical for Kubernetes readiness probes.
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

// TestAPI_Todos_FilterByCompleted verifies that GET /api/v1/todos?completed=true returns only completed todos.
func TestAPI_Todos_FilterByCompleted(t *testing.T) {
	db := setupTestDB(t)
	r := router.SetupRouter(db)
	srv := httptest.NewServer(r)
	defer srv.Close()

	client := srv.Client()
	base := srv.URL + "/api/v1/todos"

	// Create 3 todos
	titles := []string{"Todo A", "Todo B", "Todo C"}
	ids := make([]float64, 0, 3)
	for _, title := range titles {
		body := fmt.Sprintf(`{"title":%q}`, title)
		resp, err := client.Post(base, "application/json", bytes.NewBufferString(body))
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var created map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
		resp.Body.Close()
		ids = append(ids, created["id"].(float64))
	}

	// Mark "Todo B" (index 1) as completed
	putBody := `{"title":"Todo B","completed":true}`
	putURL := fmt.Sprintf("%s/%.0f", base, ids[1])
	putReq, err := http.NewRequest(http.MethodPut, putURL, bytes.NewBufferString(putBody))
	require.NoError(t, err)
	putReq.Header.Set("Content-Type", "application/json")
	putResp, err := client.Do(putReq)
	require.NoError(t, err)
	putResp.Body.Close()
	require.Equal(t, http.StatusOK, putResp.StatusCode)

	// GET ?completed=true
	getResp, err := client.Get(base + "?completed=true")
	require.NoError(t, err)
	defer getResp.Body.Close()
	assert.Equal(t, http.StatusOK, getResp.StatusCode)

	var todos []map[string]interface{}
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&todos))
	require.Len(t, todos, 1)
	assert.Equal(t, "Todo B", todos[0]["title"])
}

// TestAPI_Todos_FilterBySearch verifies that GET /api/v1/todos?search=buy returns matching todos (case-insensitive).
func TestAPI_Todos_FilterBySearch(t *testing.T) {
	db := setupTestDB(t)
	r := router.SetupRouter(db)
	srv := httptest.NewServer(r)
	defer srv.Close()

	client := srv.Client()
	base := srv.URL + "/api/v1/todos"

	for _, title := range []string{"Buy groceries", "Write tests"} {
		body := fmt.Sprintf(`{"title":%q}`, title)
		resp, err := client.Post(base, "application/json", bytes.NewBufferString(body))
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
	}

	getResp, err := client.Get(base + "?search=buy")
	require.NoError(t, err)
	defer getResp.Body.Close()
	assert.Equal(t, http.StatusOK, getResp.StatusCode)

	var todos []map[string]interface{}
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&todos))
	require.Len(t, todos, 1)
	assert.Equal(t, "Buy groceries", todos[0]["title"])
}

// TestAPI_Todos_FilterByDueDate verifies that GET /api/v1/todos?due_before=<date> returns only earlier todos.
func TestAPI_Todos_FilterByDueDate(t *testing.T) {
	db := setupTestDB(t)
	r := router.SetupRouter(db)
	srv := httptest.NewServer(r)
	defer srv.Close()

	client := srv.Client()
	base := srv.URL + "/api/v1/todos"

	// Two todos with different due dates
	earlyDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	lateDate := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	middleDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	earlyBody := fmt.Sprintf(`{"title":"Early Todo","due_date":%q}`, earlyDate.Format(time.RFC3339))
	lateBody := fmt.Sprintf(`{"title":"Late Todo","due_date":%q}`, lateDate.Format(time.RFC3339))

	for _, body := range []string{earlyBody, lateBody} {
		resp, err := client.Post(base, "application/json", bytes.NewBufferString(body))
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
	}

	// Filter: due_before=middleDate → should return only "Early Todo"
	dueBefore := middleDate.Format(time.RFC3339)
	getResp, err := client.Get(base + "?due_before=" + dueBefore)
	require.NoError(t, err)
	defer getResp.Body.Close()
	assert.Equal(t, http.StatusOK, getResp.StatusCode)

	var todos []map[string]interface{}
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&todos))
	require.Len(t, todos, 1)
	assert.Equal(t, "Early Todo", todos[0]["title"])
}

// TestAPI_ClearCompleted verifies that POST /api/v1/todos/clear-completed soft-deletes all completed todos.
func TestAPI_ClearCompleted(t *testing.T) {
	db := setupTestDB(t)
	r := router.SetupRouter(db)
	srv := httptest.NewServer(r)
	defer srv.Close()

	client := srv.Client()
	base := srv.URL + "/api/v1/todos"

	// Create 3 todos
	titles := []string{"Keep this", "Done 1", "Done 2"}
	ids := make([]float64, 0, 3)
	for _, title := range titles {
		body := fmt.Sprintf(`{"title":%q}`, title)
		resp, err := client.Post(base, "application/json", bytes.NewBufferString(body))
		require.NoError(t, err)
		var created map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
		resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		ids = append(ids, created["id"].(float64))
	}

	// Mark "Done 1" and "Done 2" (indices 1 and 2) as completed
	for i, title := range []string{"Done 1", "Done 2"} {
		putBody := fmt.Sprintf(`{"title":%q,"completed":true}`, title)
		putURL := fmt.Sprintf("%s/%.0f", base, ids[i+1])
		putReq, err := http.NewRequest(http.MethodPut, putURL, bytes.NewBufferString(putBody))
		require.NoError(t, err)
		putReq.Header.Set("Content-Type", "application/json")
		putResp, err := client.Do(putReq)
		require.NoError(t, err)
		putResp.Body.Close()
		require.Equal(t, http.StatusOK, putResp.StatusCode)
	}

	// POST /api/v1/todos/clear-completed → 204
	clearResp, err := client.Post(base+"/clear-completed", "application/json", nil)
	require.NoError(t, err)
	clearResp.Body.Close()
	assert.Equal(t, http.StatusNoContent, clearResp.StatusCode)

	// GET /api/v1/todos → only 1 non-deleted todo remains
	getResp, err := client.Get(base)
	require.NoError(t, err)
	defer getResp.Body.Close()
	assert.Equal(t, http.StatusOK, getResp.StatusCode)

	var remaining []map[string]interface{}
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&remaining))
	require.Len(t, remaining, 1)
	assert.Equal(t, "Keep this", remaining[0]["title"])

	// GET /api/v1/todos/deleted → 2 soft-deleted todos
	deletedResp, err := client.Get(base + "/deleted")
	require.NoError(t, err)
	defer deletedResp.Body.Close()
	assert.Equal(t, http.StatusOK, deletedResp.StatusCode)

	var deleted []map[string]interface{}
	require.NoError(t, json.NewDecoder(deletedResp.Body).Decode(&deleted))
	assert.Len(t, deleted, 2)
}
