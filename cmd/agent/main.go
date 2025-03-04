package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/agent"
)

func main() {
	computingPower := getEnvAsInt("COMPUTING_POWER", 1)

	for i := 0; i < computingPower; i++ {
		go agent.StartAgent()
	}

	fmt.Println("Agent started")
	select {}
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
