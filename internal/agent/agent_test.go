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
	// Тестовые случаи
	tests := []struct {
		name           string
		task           models.Task
		expectedResult float64
		expectError    bool
	}{
		{
			name: "Addition task",
			task: models.Task{
				ID:            "task-add",
				ExpressionID:  "expr-add",
				Arg1:          "2",
				Arg2:          "3",
				Operation:     "+",
				OperationTime: 1000,
				Status:        "pending",
			},
			expectedResult: 5,
			expectError:    false,
		},
		{
			name: "Subtraction task",
			task: models.Task{
				ID:            "task-sub",
				ExpressionID:  "expr-sub",
				Arg1:          "5",
				Arg2:          "3",
				Operation:     "-",
				OperationTime: 1000,
				Status:        "pending",
			},
			expectedResult: 2,
			expectError:    false,
		},
		{
			name: "Multiplication task",
			task: models.Task{
				ID:            "task-mul",
				ExpressionID:  "expr-mul",
				Arg1:          "4",
				Arg2:          "3",
				Operation:     "*",
				OperationTime: 2000,
				Status:        "pending",
			},
			expectedResult: 12,
			expectError:    false,
		},
		{
			name: "Division task",
			task: models.Task{
				ID:            "task-div",
				ExpressionID:  "expr-div",
				Arg1:          "10",
				Arg2:          "2",
				Operation:     "/",
				OperationTime: 2000,
				Status:        "pending",
			},
			expectedResult: 5,
			expectError:    false,
		},
		{
			name: "Invalid operation",
			task: models.Task{
				ID:            "task-invalid-op",
				ExpressionID:  "expr-invalid-op",
				Arg1:          "10",
				Arg2:          "2",
				Operation:     "invalid",
				OperationTime: 1000,
				Status:        "pending",
			},
			expectedResult: 0,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем тестовый сервер, который возвращает задачу
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.task)
			}))
			defer ts.Close()

			// Сохранение оригинального URL и восстановление его после завершения теста
			oldURL := taskURL
			defer func() { taskURL = oldURL }()

			// Замена URL на тестовый
			taskURL = ts.URL

			go StartAgent()

			time.Sleep(3 * time.Second)

			mu.Lock()
			defer mu.Unlock()
			result, exists := TaskResults[tt.task.ID]

			if tt.expectError {
				if exists {
					t.Errorf("Expected error for task %s, but got result: %v", tt.task.ID, result)
				}
			} else {
				if !exists {
					t.Errorf("Task result for %s was not saved", tt.task.ID)
				} else if result != tt.expectedResult {
					t.Errorf("Expected result %v, got %v for task %s", tt.expectedResult, result, tt.task.ID)
				}
			}
		})
	}
}
