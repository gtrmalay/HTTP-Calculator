package orchestrator

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/handlers"
	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/middleware"
	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/storage"
)

func StartServer(connStr string) *http.Server {
	// Инициализация хранилища
	store, err := storage.NewPostgresStorage(connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Настройка маршрутов
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/register", handlers.RegisterHandler(store))
	mux.HandleFunc("/api/v1/login", handlers.LoginHandler(store))
	mux.Handle("/api/v1/calculate", middleware.AuthMiddleware(http.HandlerFunc(handlers.ExpressionHandler(store))))
	mux.Handle("/api/v1/expressions", middleware.AuthMiddleware(http.HandlerFunc(handlers.GetExpressionsHandler(store))))
	mux.Handle("/api/v1/expressions/", middleware.AuthMiddleware(http.HandlerFunc(handlers.GetExpressionByIDHandler(store))))
	mux.Handle("/internal/task", middleware.AuthMiddleware(http.HandlerFunc(handlers.GetTaskHandler(store))))
	mux.Handle("/internal/task/", middleware.AuthMiddleware(http.HandlerFunc(handlers.GetTaskByIDHandler(store))))
	mux.Handle("/internal/task/requeue", middleware.AuthMiddleware(http.HandlerFunc(handlers.RequeueTaskHandler(store))))

	// Статические файлы
	fs := http.FileServer(http.Dir("styles"))
	mux.Handle("/styles/", http.StripPrefix("/styles/", fs))

	// Запуск сервера
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		log.Println("Server running on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Инициализация очереди задач из базы
	if err := initTaskQueue(store); err != nil {
		log.Fatalf("Failed to init task queue: %v", err)
	}

	return server
}

func initTaskQueue(store *storage.PostgresStorage) error {
	ctx := context.Background()
	pendingTasks, err := store.GetPendingTasks(ctx)
	if err != nil {
		return err
	}

	for _, task := range pendingTasks {
		completed, err := store.CheckDependenciesCompleted(ctx, task.ID)
		if err != nil {
			log.Printf("Error checking dependencies for task %s: %v", task.ID, err)
			continue
		}
		if completed {
			if err := store.AddTaskToQueue(ctx, task.ID); err != nil {
				log.Printf("Failed to add task %s to queue: %v", task.ID, err)
			} else {
				log.Printf("Restored task %s to queue", task.ID)
			}
		}
	}
	return nil
}

func ShutdownServer(server *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown failed: %v", err)
	}
}
