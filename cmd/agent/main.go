package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/agent"
)

func registerUser(username, password, baseURL string) error {
	client := &http.Client{}
	data := struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}{
		Login:    username,
		Password: password,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", baseURL+"/api/v1/register", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to register user: %s", resp.Status)
		
		return nil
	}

	log.Printf("User %s registered successfully", username)
	return nil
}

func main() {
	
	username := "agent"
	password := "agent_pass"
	baseURL := "http://localhost:8080"

	
	if err := registerUser(username, password, baseURL); err != nil {
		log.Fatalf("Failed to register user: %v", err)
	}

	
	ag, err := agent.NewAgent(username, password, baseURL)
	if err != nil {
		log.Fatalf("Failed to initialize agent: %v", err)
	}

	if err := ag.Start(); err != nil {
		log.Fatalf("Failed to start agent: %v", err)
	}
	defer ag.Stop()

	
	<-make(chan struct{})
}
