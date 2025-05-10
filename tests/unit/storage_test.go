package unit

import (
	"context"
	"database/sql"
	"testing"

	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/models"
	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/storage"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func setupTestDB(t *testing.T) (*storage.PostgresStorage, func()) {
	// Настройте строку подключения к вашей тестовой базе
	connStr := "user=postgres dbname=test_calculator_db password=Ebds777staX sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	assert.NoError(t, err, "Failed to connect to test database")

	// Очищаем таблицы перед тестом
	_, err = db.Exec(`
		DROP TABLE IF EXISTS task_queue, tasks, expressions, users CASCADE;
		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			login TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE expressions (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id),
			expression TEXT NOT NULL,
			result DOUBLE PRECISION,
			status TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE tasks (
			id TEXT PRIMARY KEY,
			expression_id INTEGER REFERENCES expressions(id),
			arg1 TEXT NOT NULL,
			arg2 TEXT NOT NULL,
			operation TEXT NOT NULL,
			operation_time INTEGER NOT NULL,
			status TEXT NOT NULL,
			result DOUBLE PRECISION,
			depends_on TEXT[]
		);
		CREATE TABLE task_queue (
			task_id TEXT PRIMARY KEY REFERENCES tasks(id)
		);
	`)
	assert.NoError(t, err, "Failed to create tables")

	store, err := storage.NewPostgresStorage(connStr)
	assert.NoError(t, err, "Failed to initialize storage")

	// Функция очистки
	cleanup := func() {
		_, _ = db.Exec(`DROP TABLE IF EXISTS task_queue, tasks, expressions, users CASCADE;`)
		store.Close()
	}

	return store, cleanup
}

func TestCreateUser(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	user := &models.User{
		Login:        "testuser",
		PasswordHash: "hashedpassword",
	}

	err := store.CreateUser(ctx, user)
	assert.NoError(t, err)
	assert.NotZero(t, user.ID, "User ID should be set after creation")

	fetchedUser, err := store.GetUserByLogin(ctx, "testuser")
	assert.NoError(t, err)
	assert.Equal(t, user.ID, fetchedUser.ID, "Fetched user ID should match")
}
