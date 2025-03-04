package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/models"
)

func TestStartAgent(t *testing.T) {
	// Создаем тестовый сервер, который будет возвращать задачу
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		task := models.Task{
			ID:            "test-task-id",
			ExpressionID:  "test-expr-id",
			Arg1:          "2",
			Arg2:          "2",
			Operation:     "+",
			OperationTime: 1000,
			Status:        "pending",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(task)
	}))
	defer ts.Close()

	// Сохраняем оригинальный URL и восстанавливаем его после завершения теста
	oldURL := taskURL
	defer func() { taskURL = oldURL }() // Восстанавливаем оригинальный URL

	// Заменяем URL на тестовый
	taskURL = ts.URL

	// Запускаем агента
	go StartAgent()

	// Ждем некоторое время для выполнения задачи
	time.Sleep(2 * time.Second)

	// Проверяем, что результат задачи сохранен
	mu.Lock()
	defer mu.Unlock()
	if _, exists := TaskResults["test-task-id"]; !exists {
		t.Errorf("Task result was not saved")
	}
}
