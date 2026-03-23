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

func TestTodoRepository_GetAll_Filters(t *testing.T) {
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	// Общий seed для всех подтестов — сеется один раз в каждом t.Run.
	// Структура seed покрывает все кейсы:
	//   - "Buy groceries": active, no due_date
	//   - "Buy milk":      active, due=past
	//   - "Write tests":   completed, due=future
	//   - "Test feature":  active, due=future
	type seedTodo struct {
		title     string
		completed bool
		dueDate   *time.Time
	}
	seed := []seedTodo{
		{"Buy groceries", false, nil},
		{"Buy milk", false, timePtr(past)},
		{"Write tests", true, timePtr(future)},
		{"Test feature", false, timePtr(future)},
	}

	tests := []struct {
		name       string
		filter     model.TodoFilter
		wantLen    int
		wantTitles []string // если задано — проверяем точный набор заголовков
	}{
		{
			name:       "completed=true",
			filter:     model.TodoFilter{Completed: boolPtr(true)},
			wantLen:    1,
			wantTitles: []string{"Write tests"},
		},
		{
			name:    "completed=false",
			filter:  model.TodoFilter{Completed: boolPtr(false)},
			wantLen: 3,
		},
		{
			name:       "search case-insensitive «buy»",
			filter:     model.TodoFilter{Search: "buy"},
			wantLen:    2,
			wantTitles: []string{"Buy groceries", "Buy milk"},
		},
		{
			name:       "search «test» matches title and body",
			filter:     model.TodoFilter{Search: "test"},
			wantLen:    2,
			wantTitles: []string{"Write tests", "Test feature"},
		},
		{
			name:       "DueBefore=now returns only past task",
			filter:     model.TodoFilter{DueBefore: timePtr(now)},
			wantLen:    1,
			wantTitles: []string{"Buy milk"},
		},
		{
			name:    "DueAfter=now returns future tasks",
			filter:  model.TodoFilter{DueAfter: timePtr(now)},
			wantLen: 2,
		},
		{
			name:    "no filters returns all",
			filter:  model.TodoFilter{},
			wantLen: 4,
		},
		{
			name: "combined: completed=false + search «test»",
			filter: model.TodoFilter{
				Completed: boolPtr(false),
				Search:    "test",
			},
			wantLen:    1,
			wantTitles: []string{"Test feature"},
		},
		{
			name: "combined: search «buy» + DueBefore=now",
			filter: model.TodoFilter{
				Search:    "buy",
				DueBefore: timePtr(now),
			},
			wantLen:    1,
			wantTitles: []string{"Buy milk"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, db := setupRepoDB(t)

			// Seed
			var todos []model.Todo
			for _, s := range seed {
				todos = append(todos, model.Todo{
					Title:     s.title,
					Completed: s.completed,
					DueDate:   s.dueDate,
				})
			}
			require.NoError(t, db.Create(&todos).Error)

			// Act
			result, err := repo.GetAll(context.Background(), tt.filter)
			require.NoError(t, err)

			// Assert length
			assert.Len(t, result, tt.wantLen)

			// Assert exact titles (if specified)
			if len(tt.wantTitles) > 0 {
				got := make([]string, len(result))
				for i, r := range result {
					got[i] = r.Title
				}
				for _, want := range tt.wantTitles {
					assert.Contains(t, got, want)
				}
			}
		})
	}
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
