package main

import (
	"os"

	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/orchestrator"
)

func main() {
	connStr := os.Getenv("DB_CONN_STR")
	if connStr == "" {
		connStr = "user=postgres dbname=calculator_db password=Ebds777staX sslmode=disable"
	}

	server := orchestrator.StartServer(connStr)

	// Ждём сигнала для завершения (например, Ctrl+C)
	<-make(chan struct{})
	orchestrator.ShutdownServer(server)
}
