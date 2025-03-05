package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/models"
)

var (
	expressions = make(map[string]*models.Expression)
	tasks       = make(map[string]*models.Task)
	taskQueue   = make([]string, 0)
	mu          sync.Mutex
)

func getOperationTime(op string) int {
	switch op {
	case "+":
		return getEnvAsInt("TIME_ADDITION_MS", 1000)
	case "-":
		return getEnvAsInt("TIME_SUBTRACTION_MS", 1000)
	case "*":
		return getEnvAsInt("TIME_MULTIPLICATION_MS", 2000)
	case "/":
		return getEnvAsInt("TIME_DIVISION_MS", 2000)
	default:
		return 1000
	}
}

func getEnvAsInt(name string, defaultValue int) int {
	val, exists := os.LookupEnv(name)
	if !exists {
		return defaultValue
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return intVal
}

// Преобразует инфиксное выражение в ОПЗ (обратную польскую запись)
func infixToRPN(expression string) ([]string, error) {
	var output []string
	var stack []string

	// Приоритет операторов
	precedence := map[string]int{
		"+": 1,
		"-": 1,
		"*": 2,
		"/": 2,
	}

	for i := 0; i < len(expression); i++ {
		char := string(expression[i])

		switch {
		case char == " ":
			continue
		case isNum(char):
			// Если символ — число, считываем всё число
			num := ""
			for i < len(expression) && (unicode.IsDigit(rune(expression[i])) || expression[i] == '.') {
				num += string(expression[i])
				i++
			}
			i-- // Возвращаемся на один символ назад
			output = append(output, num)
		case char == "(":
			stack = append(stack, char)
		case char == ")":
			// Выталкиваем все операторы из стека до открывающей скобки
			for len(stack) > 0 && stack[len(stack)-1] != "(" {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			if len(stack) == 0 {
				return nil, errors.New("mismatched parentheses")
			}
			stack = stack[:len(stack)-1] // Убираем "(" из стека
		case isOperation(char):
			// Выталкиваем операторы с более высоким приоритетом
			for len(stack) > 0 && precedence[stack[len(stack)-1]] >= precedence[char] {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			stack = append(stack, char)
		default:
			return nil, errors.New("invalid character in expression")
		}
	}

	// Выталкиваем оставшиеся операторы из стека
	for len(stack) > 0 {
		if stack[len(stack)-1] == "(" {
			return nil, errors.New("mismatched parentheses")
		}
		output = append(output, stack[len(stack)-1])
		stack = stack[:len(stack)-1]
	}

	return output, nil
}

/* func tokenizeExpression(expression string) ([]string, error) {
	tokens := strings.Fields(expression)
	if len(tokens) < 3 {
		return nil, errors.New("invalid expression format")
	}
	fmt.Println("Tokens:", tokens)
	return tokens, nil
} */

func createTasksFromRPN(expressionID string, tokens []string) error {
	var stack []string // Стек для хранения операндов и промежуточных результатов

	for _, token := range tokens {
		if isNum(token) {
			stack = append(stack, token)
		} else if isOperation(token) {
			if len(stack) < 2 {
				return errors.New("Not enough operands for operation")
			}

			arg2 := stack[len(stack)-1]
			arg1 := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			if token == "/" && arg2 == "0" {
				return errors.New("Division by zero")
			}

			taskID := uuid.New().String()
			dependsOn := make([]string, 0)

			if _, err := uuid.Parse(arg1); err == nil {
				dependsOn = append(dependsOn, arg1)
			}
			if _, err := uuid.Parse(arg2); err == nil {
				dependsOn = append(dependsOn, arg2)
			}

			task := &models.Task{
				ID:            taskID,
				ExpressionID:  expressionID,
				Arg1:          arg1,
				Arg2:          arg2,
				Operation:     token,
				OperationTime: getOperationTime(token),
				Status:        "pending",
				DependsOn:     dependsOn,
			}

			mu.Lock()
			tasks[taskID] = task
			if len(dependsOn) == 0 {
				taskQueue = append(taskQueue, taskID)
			}
			mu.Unlock()

			stack = append(stack, taskID)
		} else {
			return errors.New("Invalid token in expression")
		}
	}

	if len(stack) != 1 {
		return errors.New("Invalid expression format")

	}

	return nil
}

func isNum(token string) bool {
	_, err := strconv.ParseFloat(token, 64)
	return err == nil
}

// Проверка, является ли токен операцией
func isOperation(token string) bool {
	switch token {
	case "+", "-", "*", "/":
		return true
	default:
		return false
	}
}

func updateExpressionStatus(expressionID string) {
	mu.Lock()
	defer mu.Unlock()

	expr, exists := expressions[expressionID]
	if !exists {
		return
	}

	totalTasks := 0
	completedTasks := 0
	var finalResult float64

	for _, task := range tasks {
		if task.ExpressionID == expressionID {
			totalTasks++
			if task.Status == "completed" {
				completedTasks++
				finalResult = task.Result
			}
		}
	}

	if totalTasks > 0 && totalTasks == completedTasks {
		expr.Status = "completed"
		expr.Result = finalResult
		fmt.Println("Expression", expressionID, "completed. Result:", finalResult)
	}
}

///////// HANDLERS ///////////////////////////////////////////////////////

// Обрабатывает POST /api/v1/calculate
func ExpressionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var exprReq models.Expression
	err = json.Unmarshal(body, &exprReq)
	if err != nil || exprReq.Expression == "" {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	exprID := uuid.New().String()
	expressions[exprID] = &models.Expression{
		ID:         exprID,
		Expression: exprReq.Expression,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}

	tokens, err := infixToRPN(exprReq.Expression)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error": "Invalid expression format"}`, http.StatusUnprocessableEntity)
		return
	}

	err = createTasksFromRPN(exprID, tokens)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")

		errorMsg := fmt.Sprintf(`{"error": %s"}`, err.Error())
		http.Error(w, errorMsg, http.StatusUnprocessableEntity)
		return
	}

	fmt.Println("Expression created with ID:", exprID)
	resp := map[string]string{"id": exprID}
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(jsonResp)
}

// Обрабатывает GET /internal/task (отдаёт агенту задачу)
func GetTaskHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error": "Invalid request"}`, http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var taskRes models.Task
		err = json.Unmarshal(body, &taskRes)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
			return
		}

		mu.Lock()
		task, exists := tasks[taskRes.ID]
		if !exists {
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error": "Task not found"}`, http.StatusNotFound)
			return
		}

		if task.Operation == "/" && taskRes.Arg2 == "0" {
			task.Status = "error"
			mu.Unlock()

			updateExpressionStatus(task.ExpressionID)
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error": "Division by zero"}`, http.StatusUnprocessableEntity)
			return
		}

		task.Status = "completed"
		task.Result = taskRes.Result
		mu.Unlock()

		// Обновляем статус выражения
		updateExpressionStatus(task.ExpressionID)

		fmt.Println("До блока с зависимыми")
		// Добавляем зависимые задачи в очередь
		mu.Lock()
		for _, t := range tasks {
			if t.Status == "pending" && contains(t.DependsOn, taskRes.ID) {
				allDepsCompleted := true
				for _, depID := range t.DependsOn {
					if tasks[depID].Status != "completed" {
						allDepsCompleted = false
						break
					}
				}
				if allDepsCompleted {
					taskQueue = append(taskQueue, t.ID)
					fmt.Println("Task added to queue:", t.ID)
				}
			}
		}

		mu.Unlock()

		fmt.Println("Task submitted successfully:", taskRes.ID)
		w.WriteHeader(http.StatusOK)
	case http.MethodGet:
		mu.Lock()
		defer mu.Unlock()

		if len(taskQueue) == 0 {
			fmt.Println("No tasks in queue")
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error": "No tasks available"}`, http.StatusNotFound)
			return
		}

		taskID := taskQueue[0]
		taskQueue = taskQueue[1:]
		task := tasks[taskID]

		fmt.Println("Task dispatched:", taskID, "Operation:", task.Operation)
		jsonResp, err := json.Marshal(task)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonResp)
	}
}

func contains(slice []string, item string) bool {
	fmt.Println("Checking if", item, "is in", slice)
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func PrintTasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tasksJson, err := json.Marshal(tasks)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}
	w.Write(tasksJson)

	fmt.Println("All tasks printed.")
}

func PrintExpressionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	type ExprWithoutExprNTime struct {
		ID     string  `json:"id"`
		Status string  `json:"status"`
		Result float64 `json:"result"`
	}

	type ExprForOutput struct {
		Expressions []ExprWithoutExprNTime `json:"expressions"`
	}

	var expressionsForOutput []ExprWithoutExprNTime
	for _, expr := range expressions {
		expressionsForOutput = append(expressionsForOutput, ExprWithoutExprNTime{
			ID:     expr.ID,
			Status: expr.Status,
			Result: expr.Result,
		})
	}

	outputExpressions := ExprForOutput{expressionsForOutput}

	w.Header().Set("Content-Type", "application/json")
	expressionsJson, err := json.Marshal(outputExpressions)
	if err != nil {
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}
	w.Write(expressionsJson)
	fmt.Println("All tasks printed.")
}

// Обрабатывает GET /internal/task/{id} (возврат задачу по ID)
func GetTaskByIDHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	// Извлекаем ID задачи из URL
	taskID := r.URL.Path[len("/internal/task/"):]
	if taskID == "" {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error": "Task ID is required"}`, http.StatusBadRequest)
		return
	}

	// Ищем задачу по ID
	task, exists := tasks[taskID]
	if !exists {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error": "Task not found"}`, http.StatusNotFound)
		return
	}

	// Возвращаем задачу в формате JSON
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(task)
	if err != nil {
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}
}

func GetExpressionByIDHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	// Извлекаем ID задачи из URL
	expressionID := r.URL.Path[len("/api/v1/expressions/"):]
	if expressionID == "" {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error": "Task ID is required"}`, http.StatusBadRequest)
		return
	}

	// Ищем задачу по ID
	expression, exists := expressions[expressionID]
	if !exists {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error": "Task not found"}`, http.StatusNotFound)
		return
	}

	type ExprByID struct {
		ID     string  `json:"id"`
		Status string  `json:"status"`
		Result float64 `json:"result"`
	}

	type ExprByIDForOutput struct {
		Expression ExprByID `json:"expression"`
	}

	var exprByID ExprByID
	exprByID = ExprByID{
		ID:     expression.ID,
		Status: expression.Status,
		Result: expression.Result,
	}

	exprByIDForOutput := ExprByIDForOutput{exprByID}

	// Возвращаем задачу в формате JSON
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(exprByIDForOutput)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}
}

/////////HANDLERS/////////////////////////////////////////////////////////
