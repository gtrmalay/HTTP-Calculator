package main

import (
	"fmt"
	"net/http"

	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/handlers"
)

func main() {
	http.HandleFunc("/api/v1/calculate", handlers.ExpressionHandler)
	http.HandleFunc("/api/v1/expressions", handlers.PrintExpressionsHandler)
	http.HandleFunc("/api/v1/expressions/", handlers.GetExpressionByIDHandler)
	http.HandleFunc("/internal/task", handlers.GetTaskHandler)
	http.HandleFunc("/internal/task/submit", handlers.SubmitTaskHandler)
	http.HandleFunc("/internal/task/", handlers.GetTaskByIDHandler)
	http.HandleFunc("/internal/tasks", handlers.PrintTasksHandler) //all tasks for debug

	fmt.Println("Server running on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
