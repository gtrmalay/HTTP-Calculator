package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"unicode"

	"github.com/google/uuid"
	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/models"
	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/storage"
)

var (
	expressions       = make(map[string]*models.Expression)
	tasks             = make(map[string]*models.Task)
	taskQueue         = make([]string, 0)
	mu                sync.Mutex
	ErrDivisionByZero = errors.New("division by zero")
)

func ExpressionHandler(s *storage.PostgresStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int)

		var exprReq models.ExpressionRequest
		if err := json.NewDecoder(r.Body).Decode(&exprReq); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		expr := models.Expression{
			UserID:     userID,
			Expression: exprReq.Expression,
			Status:     "pending",
		}

		if err := s.CreateExpression(r.Context(), &expr); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to create expression")
			return
		}

		if err := CreateTasksFromExpression(s, &expr); err != nil {
			_ = s.DeleteExpression(r.Context(), expr.ID)

			var status int
			var msg string
			if errors.Is(err, ErrDivisionByZero) {
				status = http.StatusUnprocessableEntity
				msg = "Division by zero"
			} else {
				status = http.StatusUnprocessableEntity
				msg = fmt.Sprintf("Invalid expression: %v", err)
			}
			respondWithError(w, status, msg)
			return
		}

		respondWithJSON(w, http.StatusCreated, map[string]int{"id": expr.ID})
	}
}

func IsNum(token string) bool {
	_, err := strconv.ParseFloat(token, 64)
	return err == nil
}

func IsOperation(token string) bool {
	switch token {
	case "+", "-", "*", "/":
		return true
	default:
		return false
	}
}

func InfixToRPN(expression string) ([]string, error) {
	var output []string
	var stack []string

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
		case IsNum(char):
			num := ""
			for i < len(expression) && (unicode.IsDigit(rune(expression[i])) || expression[i] == '.') {
				num += string(expression[i])
				i++
			}
			i--
			output = append(output, num)
		case char == "(":
			stack = append(stack, char)
		case char == ")":
			for len(stack) > 0 && stack[len(stack)-1] != "(" {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			if len(stack) == 0 {
				return nil, errors.New("mismatched parentheses")
			}
			stack = stack[:len(stack)-1]
		case IsOperation(char):
			for len(stack) > 0 && precedence[stack[len(stack)-1]] >= precedence[char] {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			stack = append(stack, char)
		default:
			return nil, errors.New("invalid character in expression")
		}
	}

	for len(stack) > 0 {
		if stack[len(stack)-1] == "(" {
			return nil, errors.New("mismatched parentheses")
		}
		output = append(output, stack[len(stack)-1])
		stack = stack[:len(stack)-1]
	}

	return output, nil
}

func CreateTasksFromExpression(s *storage.PostgresStorage, expr *models.Expression) error {
	log.Println("Starting task creation for expression:", expr.Expression)

	// Преобразуем выражение в обратную польскую нотацию
	rpnTokens, err := InfixToRPN(expr.Expression)
	if err != nil {
		log.Println("RPN conversion error:", err)
		return err
	}

	var taskStack []string
	for _, token := range rpnTokens {
		if IsNum(token) {
			taskStack = append(taskStack, token)
		} else if IsOperation(token) {
			if len(taskStack) < 2 {
				return errors.New("not enough operands for operation")
			}

			arg2 := taskStack[len(taskStack)-1]
			arg1 := taskStack[len(taskStack)-2]
			taskStack = taskStack[:len(taskStack)-2]

			if token == "/" && arg2 == "0" {
				return ErrDivisionByZero
			}

			taskID := uuid.New().String()
			var dependsOn []string

			if _, err := uuid.Parse(arg1); err == nil {
				dependsOn = append(dependsOn, arg1)
			}
			if _, err := uuid.Parse(arg2); err == nil {
				dependsOn = append(dependsOn, arg2)
			}

			// Создаем задачу
			task := &models.Task{
				ID:            taskID,
				ExpressionID:  expr.ID,
				Arg1:          arg1,
				Arg2:          arg2,
				Operation:     token,
				OperationTime: GetOperationTime(token),
				Status:        "pending",
				DependsOn:     dependsOn,
			}

			// Сохраняем задачу в БД
			if err := s.CreateTask(context.Background(), task); err != nil {
				return fmt.Errorf("failed to create task: %w", err)
			}

			log.Printf("Created task %s: %s %s %s (depends on: %v)",
				task.ID, task.Arg1, task.Operation, task.Arg2, task.DependsOn)

			taskStack = append(taskStack, task.ID)
		}
	}

	if len(taskStack) != 1 {
		return errors.New("invalid expression format")
	}

	// Получаем все задачи выражения из БД
	tasks, err := s.GetTasksByExpressionID(context.Background(), expr.ID)
	if err != nil {
		log.Printf("Error getting tasks for expression %d: %v", expr.ID, err)
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	for _, task := range tasks {
		if len(task.DependsOn) == 0 {
			if err := s.AddTaskToQueue(context.Background(), task.ID); err != nil {
				log.Printf("Failed to add task %s to queue: %v", task.ID, err)
				continue
			}
			log.Printf("Task %s added to queue", task.ID)
		} else {
			log.Printf("Task %s has dependencies: %v, skipping queue", task.ID, task.DependsOn)
		}
	}

	return nil
}

func GetOperationTime(op string) int {
	switch op {
	case "+", "-":
		return 1000
	case "*", "/":
		return 2000
	default:
		return 1000
	}
}

func GetExpressionsHandler(s *storage.PostgresStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int)

		expressions, err := s.GetUserExpressions(r.Context(), userID)
		if err != nil {
			log.Printf("DB error: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to get expressions")
			return
		}

		respondWithJSON(w, http.StatusOK, expressions)
	}
}

func GetExpressionByIDHandler(s *storage.PostgresStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int)
		idStr := r.URL.Path[len("/api/v1/expressions/"):]
		id, err := strconv.Atoi(idStr)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid expression ID")
			return
		}

		expr, err := s.GetExpressionByID(r.Context(), id)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "Expression not found")
			return
		}

		if expr.UserID != userID {
			respondWithError(w, http.StatusForbidden, "Access denied")
			return
		}

		respondWithJSON(w, http.StatusOK, expr)
	}
}
