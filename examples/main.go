package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	BaseURL = "http://localhost:8020"
)

// Session represents an Axon session
type Session struct {
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
	URL       string `json:"url,omitempty"`
	Title     string `json:"title,omitempty"`
}

// NavigateRequest represents a navigation request
type NavigateRequest struct {
	URL string `json:"url"`
}

// Snapshot represents an Axon snapshot
type Snapshot struct {
	Content    string `json:"content"`
	URL        string `json:"url"`
	Title      string `json:"title"`
	State      string `json:"state"`
	TokenCount int    `json:"token_count"`
}

// ActionRequest represents an action request
type ActionRequest struct {
	Ref     string `json:"ref"`
	Action  string `json:"action"`
	Value   string `json:"value,omitempty"`
	Confirm bool   `json:"confirm"`
}

// ActionResult represents an action result
type ActionResult struct {
	Success         bool   `json:"success"`
	Result          string `json:"result,omitempty"`
	RequiresConfirm bool   `json:"requires_confirm,omitempty"`
	Message         string `json:"message,omitempty"`
}

func main() {
	fmt.Println("Axon Example: Post to X.com")
	fmt.Println("===========================")

	// Step 1: Create a session
	sessionID := "x_main"
	fmt.Printf("Creating session: %s\n", sessionID)
	
	resp, err := http.Post(BaseURL+"/api/v1/sessions", "application/json", 
		bytes.NewBuffer([]byte(fmt.Sprintf(`{"id":"%s"}`, sessionID))))
	if err != nil {
		fmt.Printf("Error creating session: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		fmt.Printf("Failed to create session: %d\n", resp.StatusCode)
		return
	}
	
	fmt.Println("Session created!")

	// Step 2: Navigate to X.com
	fmt.Println("\nNavigating to x.com...")
	
	navReq := NavigateRequest{URL: "https://x.com"}
	navBody, _ := json.Marshal(navReq)
	resp, err = http.Post(BaseURL+"/api/v1/sessions/"+sessionID+"/navigate", 
		"application/json", bytes.NewBuffer(navBody))
	if err != nil {
		fmt.Printf("Error navigating: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	var navResult struct {
		Success bool   `json:"success"`
		URL     string `json:"url"`
		Title   string `json:"title"`
	}
	json.NewDecoder(resp.Body).Decode(&navResult)
	fmt.Printf("Navigated to: %s\n", navResult.Title)

	// Step 3: Get snapshot
	fmt.Println("\nGetting snapshot...")
	resp, err = http.Post(BaseURL+"/api/v1/sessions/"+sessionID+"/snapshot", 
		"application/json", bytes.NewBuffer([]byte(`{"depth":"compact"}`)))
	if err != nil {
		fmt.Printf("Error getting snapshot: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	snapshotData, _ := io.ReadAll(resp.Body)
	var snapshot Snapshot
	json.Unmarshal(snapshotData, &snapshot)
	fmt.Printf("Snapshot (%d tokens):\n%s\n", snapshot.TokenCount, snapshot.Content)

	// Step 4: Fill the compose box (simplified - would need actual ref from snapshot)
	fmt.Println("\nFilling compose box...")
	
	actReq := ActionRequest{
		Ref:    "e1",
		Action: "fill",
		Value:  "Hello from Axon!",
	}
	actBody, _ := json.Marshal(actReq)
	resp, err = http.Post(BaseURL+"/api/v1/sessions/"+sessionID+"/act", 
		"application/json", bytes.NewBuffer(actBody))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	var actResult ActionResult
	json.NewDecoder(resp.Body).Decode(&actResult)
	fmt.Printf("Action result: %+v\n", actResult)

	// Step 5: Click post (requires confirmation)
	fmt.Println("\nPosting tweet...")
	
	actReq = ActionRequest{
		Ref:     "a1",
		Action:  "click",
		Confirm: true,
	}
	actBody, _ = json.Marshal(actReq)
	resp, err = http.Post(BaseURL+"/api/v1/sessions/"+sessionID+"/act", 
		"application/json", bytes.NewBuffer(actBody))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	json.NewDecoder(resp.Body).Decode(&actResult)
	fmt.Printf("Action result: %+v\n", actResult)

	// Wait and check status
	time.Sleep(2 * time.Second)
	
	resp, err = http.Get(BaseURL + "/api/v1/sessions/" + sessionID + "/status")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	var status struct {
		URL       string `json:"url"`
		AuthState string `json:"auth_state"`
	}
	json.NewDecoder(resp.Body).Decode(&status)
	fmt.Printf("\nFinal status - URL: %s, Auth: %s\n", status.URL, status.AuthState)

	fmt.Println("\nDone!")
}
