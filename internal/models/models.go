package models

import "time"

type Expression struct {
	ID         string    `json:"id"`
	Expression string    `json:"expression"`
	Status     string    `json:"status"`
	Result     float64   `json:"result"`
	CreatedAt  time.Time `json:"created_at"`
}

type Task struct {
	ID            string   `json:"id"`
	ExpressionID  string   `json:"expression_id"`
	Arg1          string   `json:"arg1"`
	Arg2          string   `json:"arg2"`
	Operation     string   `json:"operation"`
	OperationTime int      `json:"operation_time"`
	Status        string   `json:"status"`
	Result        float64  `json:"result"`
	DependsOn     []string `json:"depends_on"`
}
