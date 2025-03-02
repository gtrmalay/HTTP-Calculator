package handlers

import (
	"bytes"
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

func tokenizeExpression(expression string) ([]string, error) {
	tokens := tokenRegex.FindAllString(expression, -1)
	if len(tokens) < 3 {
		return nil, errors.New("invalid expression format")
	}
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
	taskQueue = append(taskQueue, taskID)
	mu.Unlock()

	return taskID, nil
}

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

	expr := expressions[expressionID]
	if expr == nil {
		return
	}

	allCompleted := true
	for _, task := range tasks {
		if task.ExpressionID == expressionID && task.Status != "completed" {
			allCompleted = false
			break
		}
	}

	if allCompleted {
		expr.Status = "completed"
	}
}

func ExpressionHandler(w http.ResponseWriter, r *http.Request) {
	for t := range tasks {
		delete(tasks, t)
	}

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

	resp := map[string]string{"id": exprID}
	jsonResp, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(jsonResp)
}

func GetTaskHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	if len(taskQueue) == 0 {
		http.Error(w, `{"error": "No tasks available"}`, http.StatusNotFound)
		return
	}

	taskID := taskQueue[0]
	taskQueue = taskQueue[1:]
	task := tasks[taskID]

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

	updateExpressionStatus(task.ExpressionID)

	w.WriteHeader(http.StatusOK)
}

func PrintExpressionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	tasksJson, _ := json.Marshal(tasks)
	w.Write(tasksJson)
}

func StartAgent() {
	for {
		time.Sleep(1 * time.Second)

		mu.Lock()
		if len(taskQueue) == 0 {
			mu.Unlock()
			continue
		}

		taskID := taskQueue[0]
		taskQueue = taskQueue[1:]
		task := tasks[taskID]
		mu.Unlock()

		allDependenciesCompleted := true
		for _, depID := range task.DependsOn {
			depTask, exists := tasks[depID]
			if !exists || depTask.Status != "completed" {
				allDependenciesCompleted = false
				break
			}
		}

		if !allDependenciesCompleted {
			mu.Lock()
			taskQueue = append(taskQueue, taskID)
			mu.Unlock()
			continue
		}

		replaceArgIfTaskID := func(arg string) (float64, error) {
			if task, exists := tasks[arg]; exists {
				if task.Status == "completed" {
					return task.Result, nil
				}
				return 0, fmt.Errorf("task %s is not completed", arg)
			}
			return strconv.ParseFloat(arg, 64)
		}

		arg1, err := replaceArgIfTaskID(task.Arg1)
		if err != nil {
			fmt.Println("Error processing Arg1:", err)
			continue
		}
		arg2, err := replaceArgIfTaskID(task.Arg2)
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
				result = 0
			} else {
				result = arg1 / arg2
			}
		}

		time.Sleep(time.Duration(task.OperationTime) * time.Millisecond)

		taskRes := models.Task{
			ID:     task.ID,
			Result: result,
			Status: "completed",
		}

		jsonRes, _ := json.Marshal(taskRes)
		resp, err := http.Post("http://localhost:8080/internal/task/submit", "application/json", bytes.NewReader(jsonRes))
		if err != nil {
			fmt.Println("Error submitting task result:", err)
			continue
		}
		resp.Body.Close()
	}
}
