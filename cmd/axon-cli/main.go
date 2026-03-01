package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

var (
	apiURL = "http://localhost:8020/api/v1"
)

func main() {
	cmd := flag.String("cmd", "", "Command to run (create, navigate, act, snapshot)")
	sessionID := flag.String("session", "default", "Session ID")
	url := flag.String("url", "", "URL to navigate to")
	action := flag.String("action", "", "Action to perform (click, fill, etc.)")
	ref := flag.String("ref", "", "Element reference ID (e.g. e1, a1)")
	value := flag.String("value", "", "Value to fill")
	confirm := flag.Bool("confirm", false, "Confirm irreversible action")

	flag.Parse()

	if *cmd == "" {
		fmt.Println("Usage: axon-cli -cmd <command> [options]")
		fmt.Println("Commands: create, navigate, act, snapshot")
		os.Exit(1)
	}

	switch *cmd {
	case "create":
		createSession(*sessionID)
	case "navigate":
		navigate(*sessionID, *url)
	case "act":
		act(*sessionID, *action, *ref, *value, *confirm)
	case "snapshot":
		snapshot(*sessionID)
	default:
		fmt.Printf("Unknown command: %s\n", *cmd)
		os.Exit(1)
	}
}

func createSession(id string) {
	reqBody, _ := json.Marshal(map[string]string{
		"id": id,
	})
	
	resp, err := http.Post(apiURL+"/sessions", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response: %s\n", string(body))
}

func navigate(id, targetURL string) {
	if targetURL == "" {
		fmt.Println("Target URL required")
		return
	}
	
	reqBody, _ := json.Marshal(map[string]string{
		"url": targetURL,
	})
	
	resp, err := http.Post(fmt.Sprintf("%s/sessions/%s/navigate", apiURL, id), "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response: %s\n", string(body))
}

func act(id, action, ref, value string, confirm bool) {
	if action == "" || ref == "" {
		fmt.Println("Action and Ref required")
		return
	}
	
	reqBody, _ := json.Marshal(map[string]interface{}{
		"action":  action,
		"ref":     ref,
		"value":   value,
		"confirm": confirm,
	})
	
	resp, err := http.Post(fmt.Sprintf("%s/sessions/%s/act", apiURL, id), "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response: %s\n", string(body))
}

func snapshot(id string) {
	reqBody, _ := json.Marshal(map[string]string{})
	resp, err := http.Post(fmt.Sprintf("%s/sessions/%s/snapshot", apiURL, id), "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response: %s\n", string(body))
}
