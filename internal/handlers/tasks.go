package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/models"
	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/storage"
)

func GetTaskHandler(s *storage.PostgresStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("GetTaskHandler called")

		task, err := s.GetNextTaskFromQueue(r.Context())
		if err != nil {
			log.Printf("Database error: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to get task")
			return
		}

		if task == nil {
			log.Println("No tasks in queue")
			respondWithError(w, http.StatusNotFound, "No tasks available")
			return
		}

		log.Printf("Returning task: %+v", task)
		respondWithJSON(w, http.StatusOK, task)
	}
}

func GetTaskByIDHandler(s *storage.PostgresStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		taskID := r.URL.Path[len("/internal/task/"):]
		if taskID == "" {
			respondWithError(w, http.StatusBadRequest, "Task ID is required")
			return
		}

		task, err := s.GetTaskByID(r.Context(), taskID)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "Task not found")
			return
		}

		respondWithJSON(w, http.StatusOK, task)
	}
}

func RequeueTaskHandler(s *storage.PostgresStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     string   `json:"id"`
			Result *float64 `json:"result"`
			Status string   `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Invalid request body: %v", err)
			respondWithError(w, http.StatusBadRequest, "Invalid request")
			return
		}

		log.Printf("RequeueTaskHandler: task %s, status %s", req.ID, req.Status)

		task, err := s.GetTaskByID(r.Context(), req.ID)
		if err != nil {
			log.Printf("Failed to get task %s: %v", req.ID, err)
			respondWithError(w, http.StatusInternalServerError, "Failed to get task")
			return
		}

		if req.Status == "completed" {
			if req.Result == nil {
				log.Printf("Result is required for completed status for task %s", req.ID)
				respondWithError(w, http.StatusBadRequest, "Result is required for completed status")
				return
			}

			// Обновляем задачу
			if err := s.UpdateTaskResult(r.Context(), req.ID, *req.Result); err != nil {
				log.Printf("Failed to update task %s: %v", req.ID, err)
				respondWithError(w, http.StatusInternalServerError, "Failed to update task")
				return
			}

			// Проверяем, завершены ли все задачи выражения
			tasks, err := s.GetTasksByExpressionID(r.Context(), task.ExpressionID)
			if err != nil {
				log.Printf("Failed to get tasks for expression %d: %v", task.ExpressionID, err)
				respondWithError(w, http.StatusInternalServerError, "Failed to get tasks")
				return
			}

			allCompleted := true
			var finalResult float64
			taskMap := make(map[string]*models.Task)
			for _, t := range tasks {
				taskMap[t.ID] = t
				if t.Status != "completed" {
					allCompleted = false
				}
			}

			if allCompleted {
				var rootTask *models.Task
				for _, t := range tasks {
					isRoot := true
					for _, other := range tasks {
						for _, dep := range other.DependsOn {
							if dep == t.ID {
								isRoot = false
								break
							}
						}
						if !isRoot {
							break
						}
					}
					if isRoot {
						rootTask = t
						break
					}
				}

				if rootTask != nil && rootTask.Result != nil {
					finalResult = *rootTask.Result
					if err := s.UpdateExpressionResult(r.Context(), task.ExpressionID, finalResult); err != nil {
						log.Printf("Failed to update expression %d: %v", task.ExpressionID, err)
						respondWithError(w, http.StatusInternalServerError, "Failed to update expression")
						return
					}
					log.Printf("Expression %d completed with result: %.2f", task.ExpressionID, finalResult)
				}
			}

			// Активируем зависимые задачи
			dependentTasks, err := s.GetDependentTasks(r.Context(), req.ID)
			if err != nil {
				log.Printf("Failed to get dependent tasks for %s: %v", req.ID, err)
			} else {
				log.Printf("Found %d dependent tasks for task %s", len(dependentTasks), req.ID)
				for _, depTask := range dependentTasks {
					completed, err := s.CheckDependenciesCompleted(r.Context(), depTask.ID)
					if err != nil {
						log.Printf("Error checking dependencies for task %s: %v", depTask.ID, err)
						continue
					}
					if completed {
						if err := s.AddTaskToQueue(r.Context(), depTask.ID); err != nil {
							log.Printf("Failed to add dependent task %s to queue: %v", depTask.ID, err)
						} else {
							log.Printf("Added dependent task %s to queue", depTask.ID)
						}
					} else {
						log.Printf("Dependencies for task %s not yet completed", depTask.ID)
					}
				}
			}
		} else if req.Status == "pending" {
			if err := s.AddTaskToQueue(r.Context(), req.ID); err != nil {
				log.Printf("Failed to requeue task %s: %v", req.ID, err)
				respondWithError(w, http.StatusInternalServerError, "Failed to requeue")
				return
			}
			log.Printf("Task %s requeued successfully", req.ID)
		}

		respondWithJSON(w, http.StatusOK, map[string]string{"status": "processed"})
	}
}
