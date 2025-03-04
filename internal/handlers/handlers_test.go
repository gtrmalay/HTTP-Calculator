package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/models"
)

func TestExpressionHandler(t *testing.T) {
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
}

func TestGetTaskHandler(t *testing.T) {
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

	// Создание ResponseRecorder. В него записывается ответ
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(GetTaskHandler)

	handler.ServeHTTP(rr, req)

	// Проверяем статус код
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Проверка тела ответа
	var task models.Task
	if err := json.Unmarshal(rr.Body.Bytes(), &task); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	// Проверка на возврат правильного ID (задачи)
	if task.ID != taskID {
		t.Errorf("handler returned unexpected task ID: got %v want %v",
			task.ID, taskID)
	}

	// Проверка на удаление задачи из очереди
	if len(taskQueue) != 0 {
		t.Errorf("task queue is not empty after dispatching task")
	}
}
