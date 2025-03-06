package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"text/template"

	"github.com/google/uuid"
	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/models"
)

func init() {
	// Загружаем шаблоны
	templates = template.Must(template.ParseGlob("../../templates/*.html"))
}

func TestExpressionHandler(t *testing.T) {
	// Тест на успешное создание выражения
	t.Run("Successful expression creation", func(t *testing.T) {
		reqBody := `{"expression": "2 + 2"}`
		req, err := http.NewRequest("POST", "/api/v1/calculate", strings.NewReader(reqBody))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(ExpressionHandler)

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusCreated)
		}

		expected := `{"id":`
		if !strings.Contains(rr.Body.String(), expected) {
			t.Errorf("handler returned unexpected body: got %v want %v",
				rr.Body.String(), expected)
		}
	})

	// Тест на пустое тело запроса
	t.Run("Empty request body", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/api/v1/calculate", strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(ExpressionHandler)

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusBadRequest)
		}
	})

	// Тест на некорректный JSON
	t.Run("Invalid JSON", func(t *testing.T) {
		reqBody := `{"expression": "2 + 2"`
		req, err := http.NewRequest("POST", "/api/v1/calculate", strings.NewReader(reqBody))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(ExpressionHandler)

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusBadRequest)
		}
	})

	// Тест на некорректное выражение
	t.Run("Invalid expression", func(t *testing.T) {
		reqBody := `{"expression": "2 + "}`
		req, err := http.NewRequest("POST", "/api/v1/calculate", strings.NewReader(reqBody))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(ExpressionHandler)

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnprocessableEntity {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusUnprocessableEntity)
		}
	})
}

func TestGetTaskHandler(t *testing.T) {
	// Тест на успешное получение задачи
	t.Run("Successful task dispatch", func(t *testing.T) {
		tasks = make(map[string]*models.Task)
		taskQueue = make([]string, 0)

		taskID := uuid.New().String()
		tasks[taskID] = &models.Task{
			ID:            taskID,
			ExpressionID:  "test-expr-id",
			Arg1:          "2",
			Arg2:          "2",
			Operation:     "+",
			OperationTime: 1000,
			Status:        "pending",
		}
		taskQueue = append(taskQueue, taskID)

		req, err := http.NewRequest("GET", "/internal/task", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(GetTaskHandler)

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		var task models.Task
		if err := json.Unmarshal(rr.Body.Bytes(), &task); err != nil {
			t.Fatalf("failed to unmarshal response body: %v", err)
		}

		if task.ID != taskID {
			t.Errorf("handler returned unexpected task ID: got %v want %v",
				task.ID, taskID)
		}

		if len(taskQueue) != 0 {
			t.Errorf("task queue is not empty after dispatching task")
		}
	})

	// Тест на пустую очередь задач
	t.Run("No tasks in queue", func(t *testing.T) {
		tasks = make(map[string]*models.Task)
		taskQueue = make([]string, 0)

		req, err := http.NewRequest("GET", "/internal/task", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(GetTaskHandler)

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusNotFound)
		}
	})

	// Тест на завершение задачи
	t.Run("Task completion", func(t *testing.T) {
		tasks = make(map[string]*models.Task)
		taskQueue = make([]string, 0)

		taskID := uuid.New().String()
		tasks[taskID] = &models.Task{
			ID:            taskID,
			ExpressionID:  "test-expr-id",
			Arg1:          "2",
			Arg2:          "2",
			Operation:     "+",
			OperationTime: 1000,
			Status:        "pending",
		}
		taskQueue = append(taskQueue, taskID)

		req, err := http.NewRequest("POST", "/internal/task", strings.NewReader(`{"id": "`+taskID+`", "result": 4}`))
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(GetTaskHandler)

		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		if tasks[taskID].Status != "completed" {
			t.Errorf("task status is not completed: got %v want %v",
				tasks[taskID].Status, "completed")
		}
	})
}
