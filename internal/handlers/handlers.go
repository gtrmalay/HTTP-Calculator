package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"

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

var tokenRegex = regexp.MustCompile(`\d+|\+|\-|\*|\/`)

// Токенизация
func tokenizeExpression(expression string) ([]string, error) {
	tokens := tokenRegex.FindAllString(expression, -1)
	if len(tokens) < 3 {
		return nil, errors.New("invalid expression format")
	}
	fmt.Println("Tokens:", tokens)
	return tokens, nil
}

func createTaskRecursive(expressionID string, tokens []string) (string, error) {
	if len(tokens) == 1 {
		return tokens[0], nil
	}

	opIndex := -1
	for i, token := range tokens {
		if token == "+" || token == "-" {
			opIndex = i
			break
		} else if token == "*" || token == "/" {
			if opIndex == -1 {
				opIndex = i
			}
		}
	}

	if opIndex == -1 {
		return "", errors.New("invalid expression format")
	}

	taskID := uuid.New().String()
	arg1, err := createTaskRecursive(expressionID, tokens[:opIndex])
	if err != nil {
		return "", err
	}
	arg2, err := createTaskRecursive(expressionID, tokens[opIndex+1:])
	if err != nil {
		return "", err
	}

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
		Operation:     tokens[opIndex],
		OperationTime: getOperationTime(tokens[opIndex]),
		Status:        "pending",
		DependsOn:     dependsOn,
	}

	mu.Lock()
	tasks[taskID] = task
	if len(dependsOn) == 0 {
		taskQueue = append(taskQueue, taskID)
		fmt.Println("Task added to queue immediately:", taskID)
	}
	mu.Unlock()

	fmt.Println("Task created:", taskID, "Operation:", task.Operation, "Depends on:", dependsOn)
	return taskID, nil
}

// Разбиение выражение на задачи
func createTasks(expressionID, expr string) error {
	tokens, err := tokenizeExpression(expr)
	if err != nil {
		return err
	}

	_, err = createTaskRecursive(expressionID, tokens)
	return err
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

/////////HANDLERS/////////////////////////////////////////////////////////

// POST /api/v1/calculate
func ExpressionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var exprReq models.Expression
	err = json.Unmarshal(body, &exprReq)
	if err != nil || exprReq.Expression == "" {
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

	err = createTasks(exprID, exprReq.Expression)
	if err != nil {
		http.Error(w, `{"error": "Failed to create tasks"}`, http.StatusUnprocessableEntity)
		return
	}

	fmt.Println("Expression created with ID:", exprID)
	resp := map[string]string{"id": exprID}
	jsonResp, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(jsonResp)
}

// GET /internal/task (отдаёт агенту задачу)
func GetTaskHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	if len(taskQueue) == 0 {
		fmt.Println("No tasks in queue")
		http.Error(w, `{"error": "No tasks available"}`, http.StatusNotFound)
		return
	}

	taskID := taskQueue[0]
	taskQueue = taskQueue[1:]
	task := tasks[taskID]

	fmt.Println("Task dispatched:", taskID, "Operation:", task.Operation)
	jsonResp, _ := json.Marshal(task)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResp)
}

func SubmitTaskHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error": "Invalid request"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var taskRes models.Task
	err = json.Unmarshal(body, &taskRes)
	if err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	mu.Lock()
	task, exists := tasks[taskRes.ID]
	if !exists {
		mu.Unlock()
		http.Error(w, `{"error": "Task not found"}`, http.StatusNotFound)
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
			taskQueue = append(taskQueue, t.ID)
			fmt.Println("Task added to queue:", t.ID)
			fmt.Println("Task status:", t.Status)
			fmt.Println("Task depends on:", t.DependsOn)
		}
	}
	mu.Unlock()

	fmt.Println("Task submitted successfully:", taskRes.ID)
	w.WriteHeader(http.StatusOK)
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
	tasksJson, _ := json.Marshal(tasks)
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
	expressionsJson, _ := json.Marshal(outputExpressions)
	w.Write(expressionsJson)
	fmt.Println("All tasks printed.")
}

// GET /internal/task/{id} (возврат задачу по ID)
func GetTaskByIDHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	// Извлекаем ID задачи из URL
	taskID := r.URL.Path[len("/internal/task/"):]
	if taskID == "" {
		http.Error(w, `{"error": "Task ID is required"}`, http.StatusBadRequest)
		return
	}

	// Ищем задачу по ID
	task, exists := tasks[taskID]
	if !exists {
		http.Error(w, `{"error": "Task not found"}`, http.StatusNotFound)
		return
	}

	// Возвращаем задачу в формате JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func GetExpressionByIDHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	// Извлекаем ID задачи из URL
	expressionID := r.URL.Path[len("/api/v1/expressions/"):]
	if expressionID == "" {
		http.Error(w, `{"error": "Task ID is required"}`, http.StatusBadRequest)
		return
	}

	// Ищем задачу по ID
	expression, exists := expressions[expressionID]
	if !exists {
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
	json.NewEncoder(w).Encode(exprByIDForOutput)
}

/////////HANDLERS/////////////////////////////////////////////////////////
