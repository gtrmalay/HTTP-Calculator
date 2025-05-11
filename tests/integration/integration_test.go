package integration

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/agent"
	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/orchestrator"
	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/storage"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

const (
	testUser     = "testuser"
	testPassword = "testpass"
)

var db *sql.DB

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

func setup() {
	os.MkdirAll("internal/handlers/templates", 0755)
	os.WriteFile("internal/handlers/templates/base.html", []byte(`<html></html>`), 0644)
}

func teardown() {
	os.RemoveAll("internal/handlers/templates")
}

func TestIntegration(t *testing.T) {
	connStr := "user=postgres dbname=test_calculator_db password=Ebds777staX sslmode=disable"
	if envConnStr := os.Getenv("TEST_DB_CONN_STR"); envConnStr != "" {
		connStr = envConnStr
	}

	var err error
	db, err = sql.Open("postgres", connStr)
	require.NoError(t, err)
	defer db.Close()

	err = initDatabaseSchema(db)
	require.NoError(t, err)

	err = clearDatabase(db)
	require.NoError(t, err)

	store, err := storage.NewPostgresStorage(connStr)
	require.NoError(t, err)
	defer store.Close()

	orchestratorServer := orchestrator.StartServer(connStr)
	defer orchestrator.ShutdownServer(orchestratorServer)

	time.Sleep(2 * time.Second)

	err = registerUser(db)
	require.NoError(t, err)

	token, err := loginUser()
	require.NoError(t, err)

	ag, err := agent.NewAgent(testUser, testPassword, "http://localhost:8080")
	require.NoError(t, err)
	require.NoError(t, ag.Start())
	defer ag.Stop()

	time.Sleep(3 * time.Second)

	t.Run("Simple expression calculation", func(t *testing.T) {
		exprID, err := submitExpression(token, "2+3")
		require.NoError(t, err)

		time.Sleep(5 * time.Second)

		var result float64
		err = db.QueryRow("SELECT result FROM expressions WHERE id = $1 AND user_id = (SELECT id FROM users WHERE login = $2)", exprID, testUser).Scan(&result)
		require.NoError(t, err)
		assert.Equal(t, 5.0, result)
	})

	t.Run("Complex expression with dependencies", func(t *testing.T) {
		exprID, err := submitExpression(token, "(2+3)*4")
		require.NoError(t, err)

		time.Sleep(10 * time.Second)

		var result float64
		err = db.QueryRow("SELECT result FROM expressions WHERE id = $1 AND user_id = (SELECT id FROM users WHERE login = $2)", exprID, testUser).Scan(&result)
		require.NoError(t, err)
		assert.Equal(t, 20.0, result)
	})

	t.Run("Error handling", func(t *testing.T) {
		resp, err := submitExpressionWithError(token, "2++3")
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

		resp, err = submitExpressionWithError(token, "5/0")
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	})
}

func initDatabaseSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			login TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE TABLE IF NOT EXISTS expressions (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id),
			expression TEXT NOT NULL,
			result FLOAT,
			status TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			expression_id INTEGER REFERENCES expressions(id),
			arg1 TEXT NOT NULL,
			arg2 TEXT NOT NULL,
			operation TEXT NOT NULL,
			operation_time INTEGER NOT NULL,
			status TEXT NOT NULL,
			result FLOAT,
			depends_on TEXT[]
		);
		
		CREATE TABLE IF NOT EXISTS task_queue (
			task_id TEXT PRIMARY KEY REFERENCES tasks(id)
		);
	`)
	return err
}

func clearDatabase(db *sql.DB) error {
	tables := []string{"task_queue", "tasks", "expressions", "users"}
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", pq.QuoteIdentifier(table)))
		if err != nil {
			return err
		}
	}
	return nil
}

func registerUser(db *sql.DB) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = db.Exec(
		"INSERT INTO users (login, password_hash) VALUES ($1, $2) ON CONFLICT (login) DO UPDATE SET password_hash = $2",
		testUser,
		string(hashedPassword),
	)
	return err
}

func loginUser() (string, error) {
	reqBody := map[string]string{
		"login":    testUser,
		"password": testPassword,
	}
	jsonBody, _ := json.Marshal(reqBody)

	resp, err := http.Post("http://localhost:8080/api/v1/login", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed with status: %d", resp.StatusCode)
	}

	var loginResp struct {
		Token string `json:"token"`
	}
	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	if err != nil {
		return "", err
	}

	return loginResp.Token, nil
}

func submitExpression(token, expr string) (int, error) {
	reqBody := map[string]string{
		"expression": expr,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", "http://localhost:8080/api/v1/calculate", bytes.NewBuffer(jsonBody))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var exprResp struct {
		ID int `json:"id"`
	}
	err = json.Unmarshal(body, &exprResp)
	if err != nil {
		return 0, err
	}

	return exprResp.ID, nil
}

func submitExpressionWithError(token, expr string) (*http.Response, error) {
	reqBody := map[string]string{
		"expression": expr,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", "http://localhost:8080/api/v1/calculate", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	return client.Do(req)
}
