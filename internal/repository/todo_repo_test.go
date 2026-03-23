package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Subcore/todo-app-v2/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupRepoDB(t *testing.T) (TodoRepository, *gorm.DB) {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("Skipping test: TEST_DB_DSN not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&model.Todo{})
	require.NoError(t, err)

	err = db.Exec("TRUNCATE todos RESTART IDENTITY CASCADE").Error
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Exec("TRUNCATE todos RESTART IDENTITY CASCADE")
	})

	repo := NewTodoRepository(db)
	return repo, db
}

func boolPtr(b bool) *bool {
	return &b
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func TestTodoRepository_GetAll_FilterByCompleted(t *testing.T) {
	repo, db := setupRepoDB(t)

	// Seed: create 3 todo -- 2 completed, 1 active
	todos := []model.Todo{
		{Title: "Task 1", Completed: true},
		{Title: "Task 2", Completed: true},
		{Title: "Task 3", Completed: false},
	}
	require.NoError(t, db.Create(&todos).Error)

	ctx := context.Background()

	// Call: filter by completed=true
	result, err := repo.GetAll(ctx, model.TodoFilter{Completed: boolPtr(true)})
	require.NoError(t, err)

	// Assert: len == 2, all completed == true
	assert.Len(t, result, 2)
	for _, todo := range result {
		assert.True(t, todo.Completed)
	}
}

func TestTodoRepository_GetAll_FilterBySearch(t *testing.T) {
	repo, db := setupRepoDB(t)

	// Seed: "Buy groceries", "Write tests", "Buy milk"
	todos := []model.Todo{
		{Title: "Buy groceries"},
		{Title: "Write tests"},
		{Title: "Buy milk"},
	}
	require.NoError(t, db.Create(&todos).Error)

	ctx := context.Background()

	// Call
	result, err := repo.GetAll(ctx, model.TodoFilter{Search: "buy"})
	require.NoError(t, err)

	// Assert: len == 2, both contain "buy" case-insensitive
	assert.Len(t, result, 2)
	titles := []string{result[0].Title, result[1].Title}
	assert.Contains(t, titles, "Buy groceries")
	assert.Contains(t, titles, "Buy milk")
}

func TestTodoRepository_GetAll_FilterByDueBefore(t *testing.T) {
	repo, db := setupRepoDB(t)

	now := time.Now()
	past := now.Add(-24 * time.Hour)
	today := now
	future := now.Add(24 * time.Hour)

	// Seed: 3 todo with different due_date (past, today, future)
	todos := []model.Todo{
		{Title: "Past Task", DueDate: timePtr(past)},
		{Title: "Today Task", DueDate: timePtr(today)},
		{Title: "Future Task", DueDate: timePtr(future)},
	}
	require.NoError(t, db.Create(&todos).Error)

	ctx := context.Background()

	// Call: with DueBefore = today
	result, err := repo.GetAll(ctx, model.TodoFilter{DueBefore: timePtr(today)})
	require.NoError(t, err)

	// Assert: returns only the past task (due_date < today)
	require.Len(t, result, 1)
	assert.Equal(t, "Past Task", result[0].Title)
}

func TestTodoRepository_GetAll_FilterByDueAfter(t *testing.T) {
	repo, db := setupRepoDB(t)

	now := time.Now()
	past := now.Add(-24 * time.Hour)
	today := now
	future := now.Add(24 * time.Hour)

	// Seed
	todos := []model.Todo{
		{Title: "Past Task", DueDate: timePtr(past)},
		{Title: "Today Task", DueDate: timePtr(today)},
		{Title: "Future Task", DueDate: timePtr(future)},
	}
	require.NoError(t, db.Create(&todos).Error)

	ctx := context.Background()

	// Call: with DueAfter = today
	result, err := repo.GetAll(ctx, model.TodoFilter{DueAfter: timePtr(today)})
	require.NoError(t, err)

	// Assert: returns only the future task (due_date > today)
	require.Len(t, result, 1)
	assert.Equal(t, "Future Task", result[0].Title)
}

func TestTodoRepository_GetAll_CombinedFilters(t *testing.T) {
	repo, db := setupRepoDB(t)

	now := time.Now()
	future := now.Add(24 * time.Hour)

	// Seed: 4 todo with different completed + due_date + title
	todos := []model.Todo{
		{Title: "Buy test groceries", Completed: false},
		{Title: "Write test code", Completed: true},
		{Title: "Buy milk", Completed: false},
		{Title: "Test new feature", Completed: false, DueDate: timePtr(future)},
	}
	require.NoError(t, db.Create(&todos).Error)

	ctx := context.Background()

	// Call: Completed=false, Search="test"
	result, err := repo.GetAll(ctx, model.TodoFilter{
		Completed: boolPtr(false),
		Search:    "test",
	})
	require.NoError(t, err)

	// Assert: exactly 2 tasks match
	assert.Len(t, result, 2)
}

func TestTodoRepository_GetAll_NoFilters(t *testing.T) {
	repo, db := setupRepoDB(t)

	// Seed: 3 todo
	todos := []model.Todo{
		{Title: "Task 1"},
		{Title: "Task 2"},
		{Title: "Task 3"},
	}
	require.NoError(t, db.Create(&todos).Error)

	ctx := context.Background()

	// Call: empty filter
	result, err := repo.GetAll(ctx, model.TodoFilter{})
	require.NoError(t, err)

	// Assert: len == 3
	assert.Len(t, result, 3)
}

func TestTodoRepository_Delete_SoftDelete(t *testing.T) {
	repo, db := setupRepoDB(t)
	ctx := context.Background()

	// Seed: create one todo
	todo := model.Todo{Title: "Task to delete"}
	require.NoError(t, db.Create(&todo).Error)

	// Call: Delete
	err := repo.Delete(ctx, todo.ID)
	require.NoError(t, err)

	// Assert: GetByID returns ErrNotFound
	_, err = repo.GetByID(ctx, todo.ID)
	assert.ErrorIs(t, err, ErrNotFound)

	// Assert: GetDeleted contains it and it has non-empty deleted_at
	deletedTodos, err := repo.GetDeleted(ctx)
	require.NoError(t, err)
	require.Len(t, deletedTodos, 1)

	assert.Equal(t, todo.ID, deletedTodos[0].ID)
	assert.True(t, deletedTodos[0].DeletedAt.Valid)
	assert.NotZero(t, deletedTodos[0].DeletedAt.Time)
}

func TestTodoRepository_DeleteCompleted(t *testing.T) {
	repo, db := setupRepoDB(t)
	ctx := context.Background()

	// Seed: 2 completed, 1 active
	todos := []model.Todo{
		{Title: "Task 1", Completed: true},
		{Title: "Task 2", Completed: true},
		{Title: "Task 3", Completed: false},
	}
	require.NoError(t, db.Create(&todos).Error)

	// Call: DeleteCompleted
	err := repo.DeleteCompleted(ctx)
	require.NoError(t, err)

	// Assert: GetAll returns only 1 active
	remaining, err := repo.GetAll(ctx, model.TodoFilter{})
	require.NoError(t, err)
	assert.Len(t, remaining, 1)
	assert.Equal(t, "Task 3", remaining[0].Title)
	assert.False(t, remaining[0].Completed)

	// Assert: GetDeleted returns 2 tasks
	deletedTodos, err := repo.GetDeleted(ctx)
	require.NoError(t, err)
	assert.Len(t, deletedTodos, 2)
}

func TestTodoRepository_Delete_NotFound(t *testing.T) {
	repo, _ := setupRepoDB(t)
	ctx := context.Background()

	// Call: Delete non-existent ID
	err := repo.Delete(ctx, 999999)

	// Assert
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestTodoRepository_GetDeleted_Empty(t *testing.T) {
	repo, _ := setupRepoDB(t)
	ctx := context.Background()

	// Call: GetDeleted on empty DB
	deletedTodos, err := repo.GetDeleted(ctx)

	// Assert
	require.NoError(t, err)
	assert.Empty(t, deletedTodos)
}
