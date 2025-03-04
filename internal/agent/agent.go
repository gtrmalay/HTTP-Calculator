package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/models"
)

var (
	TaskResults = make(map[string]float64) // Хранилище результатов задач
	mu          sync.Mutex                 // Мьютекс для потокобезопасного доступа к `TaskResults`
)

func StartAgent() {
	for {
		time.Sleep(500 * time.Millisecond)

		// Запрашиваем задачу через GET /internal/task
		resp, err := http.Get("http://localhost:8080/internal/task")
		if err != nil {
			fmt.Println("Error fetching task:", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Println("No tasks available or server error")
			continue
		}

		var task models.Task
		if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
			fmt.Println("Error decoding task:", err)
			continue
		}

		fmt.Printf("Received task: %s | Operation: %s | Args: %s, %s\n", task.ID, task.Operation, task.Arg1, task.Arg2)

		// Проверяем, завершены ли все зависимости
		allDependenciesCompleted := true
		for _, depID := range task.DependsOn {
			resp, err := http.Get("http://localhost:8080/internal/task/" + depID)
			if err != nil {
				fmt.Println("Error fetching dependency task:", err)
				allDependenciesCompleted = false
				break
			}
			defer resp.Body.Close()

			var depTask models.Task
			if err := json.NewDecoder(resp.Body).Decode(&depTask); err != nil {
				fmt.Println("Error decoding dependency task:", err)
				allDependenciesCompleted = false
				break
			}

			fmt.Printf("Dependency task %s status: %s\n", depID, depTask.Status) // Логирование статуса
			if depTask.Status != "completed" {
				fmt.Printf("Dependency task %s is not completed\n", depID)
				allDependenciesCompleted = false
				break
			}
		}

		if !allDependenciesCompleted {
			// Возвращаем задачу в очередь
			fmt.Println("Task dependencies not completed, requeuing task:", task.ID)
			_, err := http.Post("http://localhost:8080/internal/task", "application/json", bytes.NewReader([]byte(`{"id":"`+task.ID+`"}`)))
			if err != nil {
				fmt.Println("Error requeuing task:", err)
			}
			continue
		}

		// Выполняем задачу
		arg1, err := getArgumentValue(task.Arg1)
		if err != nil {
			fmt.Println("Error processing Arg1:", err)
			continue
		}
		arg2, err := getArgumentValue(task.Arg2)
		if err != nil {
			fmt.Println("Error processing Arg2:", err)
			continue
		}

		var result float64
		switch task.Operation {
		case "+":
			result = arg1 + arg2
		case "-":
			result = arg1 - arg2
		case "*":
			result = arg1 * arg2
		case "/":
			if arg2 == 0 {
				fmt.Println("Error: Division by zero")
				result = 0
			} else {
				result = arg1 / arg2
			}
		default:
			fmt.Println("Unknown operation:", task.Operation)
			continue
		}

		// Симуляция выполнения задачи
		time.Sleep(time.Duration(task.OperationTime) * time.Millisecond)

		// Сохраняем результат
		mu.Lock()
		TaskResults[task.ID] = result
		mu.Unlock()

		fmt.Println("Task result saved:", task.ID, "Result:", result) // Логирование

		// Отправляем результат через POST /internal/task/submit
		taskRes := models.Task{
			ID:     task.ID,
			Result: result,
			Status: "completed",
		}

		fmt.Println("Submitting task result:", task.ID, "Result:", result) // Логирование
		jsonRes, _ := json.Marshal(taskRes)
		resp, err = http.Post("http://localhost:8080/internal/task/submit", "application/json", bytes.NewReader(jsonRes))
		if err != nil {
			fmt.Println("Error submitting task result:", err)
			continue
		}
		resp.Body.Close()

		fmt.Printf("Task completed: %s | Result: %.2f\n", task.ID, result)
	}
}

// getArgumentValue получает значение аргумента, если это число — парсит его, если ID задачи — берет из `TaskResults`
func getArgumentValue(arg string) (float64, error) {
	if _, err := uuid.Parse(arg); err == nil {
		// Если аргумент является ID задачи, запрашиваем результат
		mu.Lock()
		defer mu.Unlock()
		if result, exists := TaskResults[arg]; exists {
			return result, nil
		}
		return 0, fmt.Errorf("task %s result not found", arg)
	}
	// Иначе конвертируем аргумент в число
	return strconv.ParseFloat(arg, 64)
}
