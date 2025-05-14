package main

import (
	"os"

	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/orchestrator"
)

func main() {
	connStr := os.Getenv("DB_CONN_STR")
	if connStr == "" {
		connStr = "user=postgres dbname=cc password=Ebds777staX sslmode=disable"
	}

	server := orchestrator.StartServer(connStr)
	<-make(chan struct{})
	orchestrator.ShutdownServer(server)
}

/* func RunMigrations(connStr string) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("❌ Failed to connect to DB:", err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatal("❌ Failed to create driver:", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		log.Fatal("❌ Failed to create migrate instance:", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal("❌ Migration failed:", err)
	}

	log.Println("✅ Migrations applied successfully.")
} */
