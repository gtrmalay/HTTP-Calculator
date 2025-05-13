package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID           int       `json:"id"`
	Login        string    `json:"login"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Claims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

type Expression struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	Expression string    `json:"expression"`
	Result     float64   `json:"result"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

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

type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type ExpressionRequest struct {
	Expression string `json:"expression"`
}
