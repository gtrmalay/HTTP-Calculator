package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/models"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type PostgresStorage struct {
	DB *sql.DB
}

func NewPostgresStorage(connStr string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	return &PostgresStorage{DB: db}, nil
}

// User methods
func (s *PostgresStorage) CreateUser(ctx context.Context, user *models.User) error {
	return s.DB.QueryRowContext(ctx,
		"INSERT INTO users (login, password_hash) VALUES ($1, $2) RETURNING id, created_at",
		user.Login, user.PasswordHash).Scan(&user.ID, &user.CreatedAt)
}

func (s *PostgresStorage) GetUserByLogin(ctx context.Context, login string) (*models.User, error) {
	var user models.User
	err := s.DB.QueryRowContext(ctx,
		"SELECT id, login, password_hash, created_at FROM users WHERE login = $1",
		login).Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Expression methods
func (s *PostgresStorage) CreateExpression(ctx context.Context, expr *models.Expression) error {
	return s.DB.QueryRowContext(ctx,
		"INSERT INTO expressions (user_id, expression, status) VALUES ($1, $2, $3) RETURNING id, created_at",
		expr.UserID, expr.Expression, expr.Status).Scan(&expr.ID, &expr.CreatedAt)
}

func (s *PostgresStorage) GetExpressionByID(ctx context.Context, id int) (*models.Expression, error) {
	var expr models.Expression
	err := s.DB.QueryRowContext(ctx,
		"SELECT id, user_id, expression, result, status, created_at FROM expressions WHERE id = $1",
		id).Scan(&expr.ID, &expr.UserID, &expr.Expression, &expr.Result, &expr.Status, &expr.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &expr, nil
}

func (s *PostgresStorage) GetUserExpressions(ctx context.Context, userID int) ([]models.Expression, error) {
	log.Printf("Executing query for user %d", userID)

	rows, err := s.DB.QueryContext(ctx,
		"SELECT id, expression, result, status, created_at FROM expressions WHERE user_id = $1 ORDER BY created_at DESC",
		userID)
	if err != nil {
		log.Printf("Query error: %v", err)
		return nil, err
	}
	defer rows.Close()

	var expressions []models.Expression
	for rows.Next() {
		var expr models.Expression
		if err := rows.Scan(&expr.ID, &expr.Expression, &expr.Result, &expr.Status, &expr.CreatedAt); err != nil {
			return nil, err
		}
		expressions = append(expressions, expr)
	}

	return expressions, nil
}

func (s *PostgresStorage) UpdateExpressionResult(ctx context.Context, id int, result float64) error {
	_, err := s.DB.ExecContext(ctx,
		"UPDATE expressions SET result = $1, status = 'completed' WHERE id = $2",
		result, id)
	return err
}

func (s *PostgresStorage) DeleteExpression(ctx context.Context, id int) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM expressions WHERE id = $1", id)
	return err
}

// Task methods
func (s *PostgresStorage) CreateTask(ctx context.Context, task *models.Task) error {
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO tasks 
         (id, expression_id, arg1, arg2, operation, operation_time, status, result, depends_on) 
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		task.ID,
		task.ExpressionID,
		task.Arg1,
		task.Arg2,
		task.Operation,
		task.OperationTime,
		task.Status,
		task.Result,
		pq.Array(task.DependsOn),
	)
	return err
}

func (s *PostgresStorage) GetTaskByID(ctx context.Context, id string) (*models.Task, error) {
	var task models.Task
	var dependsOn pq.StringArray
	var result sql.NullFloat64

	err := s.DB.QueryRowContext(ctx,
		"SELECT id, expression_id, arg1, arg2, operation, operation_time, status, result, depends_on FROM tasks WHERE id = $1",
		id).Scan(&task.ID, &task.ExpressionID, &task.Arg1, &task.Arg2, &task.Operation, &task.OperationTime, &task.Status, &result, &dependsOn)
	if err != nil {
		return nil, err
	}

	if result.Valid {
		task.Result = &result.Float64
	} else {
		task.Result = nil
	}

	if dependsOn != nil {
		task.DependsOn = []string(dependsOn)
	} else {
		task.DependsOn = []string{}
	}

	return &task, nil
}

func (s *PostgresStorage) GetPendingTasks(ctx context.Context) ([]models.Task, error) {
	rows, err := s.DB.QueryContext(ctx,
		"SELECT id, expression_id, arg1, arg2, operation, operation_time, depends_on FROM tasks WHERE status = 'pending'")
	if err != nil {
		log.Printf("Query error: %v", err)
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var task models.Task
		var dependsOn pq.StringArray

		if err := rows.Scan(&task.ID, &task.ExpressionID, &task.Arg1, &task.Arg2, &task.Operation, &task.OperationTime, &dependsOn); err != nil {
			return nil, err
		}

		if dependsOn != nil {
			task.DependsOn = []string(dependsOn)
		} else {
			task.DependsOn = []string{}
		}
		task.Status = "pending"

		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (s *PostgresStorage) UpdateTaskResult(ctx context.Context, id string, result float64) error {
	_, err := s.DB.ExecContext(ctx,
		"UPDATE tasks SET result = $1, status = 'completed' WHERE id = $2",
		result, id)
	return err
}

func (s *PostgresStorage) GetTasksByExpressionID(ctx context.Context, expressionID int) ([]*models.Task, error) {
	query := `
        SELECT id, expression_id, arg1, arg2, operation, 
               operation_time, status, result, depends_on
        FROM tasks 
        WHERE expression_id = $1
    `

	rows, err := s.DB.QueryContext(ctx, query, expressionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		var task models.Task
		var dependsOn pq.StringArray
		var result sql.NullFloat64

		err := rows.Scan(
			&task.ID,
			&task.ExpressionID,
			&task.Arg1,
			&task.Arg2,
			&task.Operation,
			&task.OperationTime,
			&task.Status,
			&result,
			&dependsOn,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		if result.Valid {
			task.Result = &result.Float64
		} else {
			task.Result = nil
		}

		if dependsOn != nil {
			task.DependsOn = []string(dependsOn)
		} else {
			task.DependsOn = []string{}
		}
		tasks = append(tasks, &task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return tasks, nil
}

func (s *PostgresStorage) CheckDependenciesCompleted(ctx context.Context, taskID string) (bool, error) {
	var count int
	err := s.DB.QueryRowContext(ctx,
		`SELECT COUNT(*)
         FROM tasks t
         WHERE t.id = ANY(
             (SELECT UNNEST(depends_on) FROM tasks WHERE id = $1)
         ) AND t.status != 'completed'`,
		taskID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

func (s *PostgresStorage) GetDependentTasks(ctx context.Context, taskID string) ([]*models.Task, error) {
	query := `
        SELECT id, expression_id, arg1, arg2, operation, 
               operation_time, status, result, depends_on
        FROM tasks 
        WHERE $1 = ANY(depends_on) AND status = 'pending'
    `

	rows, err := s.DB.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to query dependent tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		var task models.Task
		var dependsOn pq.StringArray
		var result sql.NullFloat64

		err := rows.Scan(
			&task.ID,
			&task.ExpressionID,
			&task.Arg1,
			&task.Arg2,
			&task.Operation,
			&task.OperationTime,
			&task.Status,
			&result,
			&dependsOn,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		if result.Valid {
			task.Result = &result.Float64
		} else {
			task.Result = nil
		}

		if dependsOn != nil {
			task.DependsOn = []string(dependsOn)
		} else {
			task.DependsOn = []string{}
		}
		tasks = append(tasks, &task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return tasks, nil
}

func (s *PostgresStorage) AddTaskToQueue(ctx context.Context, taskID string) error {
	var status string
	err := s.DB.QueryRowContext(ctx,
		"SELECT status FROM tasks WHERE id = $1", taskID).Scan(&status)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	if status != "pending" {
		return fmt.Errorf("task %s has invalid status: %s", taskID, status)
	}

	_, err = s.DB.ExecContext(ctx,
		`INSERT INTO task_queue (task_id) VALUES ($1)
         ON CONFLICT (task_id) DO NOTHING`,
		taskID)
	if err != nil {
		return fmt.Errorf("failed to insert into queue: %w", err)
	}

	return nil
}

func (s *PostgresStorage) GetNextTaskFromQueue(ctx context.Context) (*models.Task, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		return nil, err
	}
	defer tx.Rollback()

	var taskID string
	err = tx.QueryRowContext(ctx, `
        SELECT task_id FROM task_queue LIMIT 1 FOR UPDATE
    `).Scan(&taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Println("No tasks found in task_queue")
			return nil, nil
		}
		log.Printf("Error selecting from task_queue: %v", err)
		return nil, err
	}

	_, err = tx.ExecContext(ctx, `
        DELETE FROM task_queue WHERE task_id = $1
    `, taskID)
	if err != nil {
		log.Printf("Error deleting from task_queue: %v", err)
		return nil, err
	}

	log.Printf("Extracted task ID from queue: %s", taskID)

	var task models.Task
	var dependsOn pq.StringArray
	var result sql.NullFloat64

	err = tx.QueryRowContext(ctx, `
        SELECT id, expression_id, arg1, arg2, operation, 
               operation_time, status, result, depends_on 
        FROM tasks WHERE id = $1`,
		taskID).Scan(
		&task.ID, &task.ExpressionID,
		&task.Arg1, &task.Arg2,
		&task.Operation, &task.OperationTime,
		&task.Status, &result,
		&dependsOn,
	)

	if err != nil {
		log.Printf("Error fetching task %s from tasks: %v", taskID, err)
		return nil, err
	}

	if result.Valid {
		task.Result = &result.Float64
	} else {
		task.Result = nil
	}

	if dependsOn != nil {
		task.DependsOn = []string(dependsOn)
	} else {
		task.DependsOn = []string{}
	}

	log.Printf("Returning task: %+v", task)
	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		return nil, err
	}

	return &task, nil
}

func (s *PostgresStorage) Close() error {
	return s.DB.Close()
}
