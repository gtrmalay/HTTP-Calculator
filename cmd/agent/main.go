package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Task struct {
	ID            string   `json:"id"`
	ExpressionID  int      `json:"expression_id"`
	Arg1          string   `json:"arg1"`
	Arg2          string   `json:"arg2"`
	Operation     string   `json:"operation"`
	OperationTime int      `json:"operation_time"`
	Status        string   `json:"status"`
	Result        *float64 `json:"result"`
	DependsOn     []string `json:"depends_on"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

var (
	token    string
	username = "agent"
	password = "agent_pass"
)

func main() {
	if err := authenticate(); err != nil {
		log.Fatalf("Failed to authenticate: %v", err)
	}

	for {
		task, err := getTask()
		if err != nil {
			log.Printf("Error getting task: %v", err)
			if isUnauthorized(err) {
				if err := authenticate(); err != nil {
					log.Printf("Failed to re-authenticate: %v", err)
					time.Sleep(5 * time.Second)
					continue
				}
				continue
			}
			time.Sleep(5 * time.Second)
			continue
		}
		if task != nil {
			log.Printf("Received task: %+v", task)
			if err := processTask(task); err != nil {
				log.Printf("Error processing task: %v", err)
				if isUnauthorized(err) {
					if err := authenticate(); err != nil {
						log.Printf("Failed to re-authenticate: %v", err)
						time.Sleep(5 * time.Second)
						continue
					}
					continue
				}
				time.Sleep(5 * time.Second)
				continue
			}
		} else {
			log.Println("No tasks available")
			time.Sleep(5 * time.Second)
		}
	}
}

func authenticate() error {
	client := &http.Client{}
	data := struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}{
		Login:    username,
		Password: password,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal login data: %w", err)
	}

	req, err := http.NewRequest("POST", "http://localhost:8080/api/v1/login", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected login status: %s, body: %s", resp.Status, string(body))
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("failed to decode login response: %w", err)
	}

	token = loginResp.Token
	log.Printf("Successfully authenticated, new token: %s", token)
	return nil
}

func getTask() (*Task, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:8080/internal/task", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status: %s, body: %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var task Task
	if err := json.Unmarshal(body, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

func processTask(task *Task) error {
	log.Printf("Processing task %s: %s %s %s", task.ID, task.Arg1, task.Operation, task.Arg2)

	arg1Value := getArgValue(task.Arg1)
	if arg1Value == -1 {
		return fmt.Errorf("failed to get value for Arg1: %s", task.Arg1)
	}
	log.Printf("Arg1 value for task %s: %.2f", task.ID, arg1Value)

	arg2Value := getArgValue(task.Arg2)
	if arg2Value == -1 {
		return fmt.Errorf("failed to get value for Arg2: %s", task.Arg2)
	}
	log.Printf("Arg2 value for task %s: %.2f", task.ID, arg2Value)

	var result float64
	switch task.Operation {
	case "+":
		result = arg1Value + arg2Value
	case "*":
		result = arg1Value * arg2Value
	default:
		return fmt.Errorf("unsupported operation: %s", task.Operation)
	}
	log.Printf("Computed result for task %s: %.2f", task.ID, result)

	time.Sleep(time.Duration(task.OperationTime) * time.Millisecond)

	if err := submitResult(task.ID, result); err != nil {
		return fmt.Errorf("failed to submit result: %w", err)
	}
	log.Printf("Successfully submitted result for task %s: %.2f", task.ID, result)

	return nil
}

func getArgValue(arg string) float64 {
	uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	if uuidRegex.MatchString(arg) {
		log.Printf("Arg %s is a task ID, fetching result", arg)
		client := &http.Client{}
		req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:8080/internal/task/%s", arg), nil)
		if err != nil {
			log.Printf("Failed to create request for task %s: %v", arg, err)
			return -1
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Failed to send request for task %s: %v", arg, err)
			return -1
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			log.Printf("Unexpected status for task %s: %s, body: %s", arg, resp.Status, string(body))
			return -1
		}

		var task Task
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Failed to read response for task %s: %v", arg, err)
			return -1
		}

		if err := json.Unmarshal(body, &task); err != nil {
			log.Printf("Failed to unmarshal response for task %s: %v", arg, err)
			return -1
		}

		if task.Result == nil || task.Status != "completed" {
			log.Printf("Task %s not completed or result unavailable", arg)
			return -1
		}

		log.Printf("Fetched result for task %s: %.2f", arg, *task.Result)
		return *task.Result
	}

	if value, err := strconv.ParseFloat(arg, 64); err == nil {
		return value
	}

	log.Printf("Invalid argument value: %s", arg)
	return -1
}

func toFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func submitResult(taskID string, result float64) error {
	data := struct {
		ID     string  `json:"id"`
		Result float64 `json:"result"`
		Status string  `json:"status"`
	}{
		ID:     taskID,
		Result: result,
		Status: "completed",
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", "http://localhost:8080/internal/task/requeue", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	log.Printf("Submitting result for task %s: %.2f", taskID, result)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %s, body: %s", resp.Status, string(body))
	}

	return nil
}

func isUnauthorized(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "401 Unauthorized")
}
