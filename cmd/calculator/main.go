package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/handlers"
)

func main() {
	http.HandleFunc("/api/v1/calculate", handlers.ExpressionHandler)
	http.HandleFunc("/internal/task/get", handlers.GetTaskHandler)
	http.HandleFunc("/internal/task/submit", handlers.SubmitTaskHandler)
	http.HandleFunc("/internal/tasks", handlers.PrintExpressionsHandler)

	computingPower := getEnvAsInt("COMPUTING_POWER", 1)

	for i := 0; i < computingPower; i++ {
		go handlers.StartAgent()
	}

	fmt.Println("Server running on :8080")
	http.ListenAndServe(":8080", nil)
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
