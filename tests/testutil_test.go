package tests

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// setupTestDB connects to Postgres using TEST_DB_DSN and cleans the database after each test.
// Skips the test if TEST_DB_DSN is not set.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN not set — skipping integration test")
		return nil
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "failed to connect to test database")

	// Confirm migrations have been applied — the todos table must exist.
	var exists bool
	err = db.Raw(`SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'todos')`).Scan(&exists).Error
	require.NoError(t, err, "failed to check table existence")
	require.True(t, exists,
		"Table 'todos' does not exist. Run migrations first:\n"+
			"  migrate -path migrations -database \"$TEST_DB_DSN\" up")

	// Cleanup database after each test
	t.Cleanup(func() {
		err := db.Exec("TRUNCATE todos RESTART IDENTITY CASCADE").Error
		require.NoError(t, err, "failed to truncate todos table")
	})

	return db
}
