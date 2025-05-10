package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/handlers"
	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/middleware"
	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/storage"
)

func main() {
	connStr := os.Getenv("DB_CONN_STR")
	if connStr == "" {
		connStr = "user=postgres dbname=calculator_db password=Ebds777staX sslmode=disable"
	}

	store, err := storage.NewPostgresStorage(connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer store.Close()

	log.Println("Restoring task queue from database...")
	pendingTasks, err := store.GetPendingTasks(context.Background())
	if err != nil {
		log.Fatalf("Failed to get pending tasks: %v", err)
	}

	for _, task := range pendingTasks {
		completed, err := store.CheckDependenciesCompleted(context.Background(), task.ID)
		if err != nil {
			log.Printf("Error checking dependencies for task %s: %v", task.ID, err)
			continue
		}
		if completed {
			if err := store.AddTaskToQueue(context.Background(), task.ID); err != nil {
				log.Printf("Failed to add task %s to queue: %v", task.ID, err)
			} else {
				log.Printf("Restored task %s to queue", task.ID)
			}
		}
	}

	// Публичные маршруты (без авторизации)
	http.HandleFunc("/api/v1/register", handlers.RegisterHandler(store))
	http.HandleFunc("/api/v1/login", handlers.LoginHandler(store))

	// Защищённые маршруты (с авторизацией)
	protected := http.NewServeMux()
	protected.HandleFunc("/api/v1/calculate", handlers.ExpressionHandler(store))
	protected.HandleFunc("/api/v1/expressions", handlers.GetExpressionsHandler(store))
	protected.HandleFunc("/api/v1/expressions/", handlers.GetExpressionByIDHandler(store))
	protected.HandleFunc("/internal/task", handlers.GetTaskHandler(store))
	protected.HandleFunc("/internal/task/", handlers.GetTaskByIDHandler(store))
	protected.HandleFunc("/internal/task/requeue", handlers.RequeueTaskHandler(store))

	// Применяем middleware только к защищённым маршрутам
	http.Handle("/api/v1/calculate", middleware.AuthMiddleware(protected))
	http.Handle("/api/v1/expressions", middleware.AuthMiddleware(protected))
	http.Handle("/api/v1/expressions/", middleware.AuthMiddleware(protected))
	http.Handle("/internal/task", middleware.AuthMiddleware(protected))
	http.Handle("/internal/task/", middleware.AuthMiddleware(protected))
	http.Handle("/internal/task/requeue", middleware.AuthMiddleware(protected))

	// Статические файлы
	fs := http.FileServer(http.Dir("styles"))
	http.Handle("/styles/", http.StripPrefix("/styles/", fs))

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
