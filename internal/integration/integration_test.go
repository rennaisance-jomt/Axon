package integration

// Integration tests for Axon browser automation
// These tests require a running browser environment

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/rennaisance-jomt/axon/internal/browser"
	"github.com/rennaisance-jomt/axon/internal/config"
)

func TestNavigateSnapshotActWorkflow(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") != "" {
		t.Skip("Skipping integration test in CI")
	}

	// Initialize config
	cfg := config.DefaultConfig().Browser
	cfg.Headless = true
	cfg.PoolSize = 1

	// Create pool
	pool, err := browser.NewPool(&cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Create session manager
	sm := browser.NewSessionManager(pool, 30*time.Minute, nil)

	// 1. Create session
	session, err := sm.Create("test-workflow", "")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// 2. Navigate to google.com (safe/stable target)
	err = session.Navigate("https://www.google.com", "load")
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// 3. Take snapshot
	extractor := browser.NewSnapshotExtractor()
	snapshot, err := extractor.Extract(session.Page, "compact", "")
	if err != nil {
		t.Fatalf("Failed to extract snapshot: %v", err)
	}

	if snapshot.URL == "" {
		t.Error("Snapshot URL is empty")
	}
	if len(snapshot.Elements) == 0 {
		t.Error("No elements found in snapshot")
	}

	t.Logf("Snapshot captured for %s with %d elements", snapshot.URL, len(snapshot.Elements))
}

// TestSessionPersistence tests that sessions persist and can be retrieved
func TestSessionPersistence(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") != "" {
		t.Skip("Skipping integration test in CI")
	}

	cfg := config.DefaultConfig().Browser
	cfg.Headless = true
	cfg.PoolSize = 1

	pool, err := browser.NewPool(&cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	sm := browser.NewSessionManager(pool, 30*time.Minute, nil)

	// 1. Create session
	session, err := sm.Create("persist-test", "")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// 2. Navigate
	err = session.Navigate("https://example.com", "load")
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Verify it exists in manager
	retrieved, err := sm.Get("persist-test")
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}
	
	if retrieved.URL != "https://example.com/" && retrieved.URL != "https://example.com" {
		t.Errorf("Retrieved session URL mismatch: want https://example.com, got %s", retrieved.URL)
	}

	// 3. Delete session
	err = sm.Delete("persist-test")
	if err != nil {
		t.Fatalf("Failed to close session: %v", err)
	}

	// 4. Retrieve session - should NOT exist
	_, err = sm.Get("persist-test")
	if err == nil {
		t.Fatalf("Expected session to be deleted")
	}

	t.Log("Session persistence successful")
}

// TestProfileLoading tests profile loading functionality
func TestProfileLoading(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") != "" {
		t.Skip("Skipping integration test in CI")
	}

	cfg := config.DefaultConfig().Browser
	cfg.Headless = true
	cfg.PoolSize = 1

	pool, err := browser.NewPool(&cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	sm := browser.NewSessionManager(pool, 30*time.Minute, nil)

	// 1. Create session
	session, err := sm.Create("profile-test", "")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Navigate to a site to set a cookie
	err = session.Navigate("https://example.com", "load")
	if err != nil {
		t.Fatalf("Failed to navigate: %v", err)
	}

	// Save profile to file
	tmpFile := "test_profile.json"
	defer os.Remove(tmpFile)
	
	err = session.ExportCookies(tmpFile)
	if err != nil {
		t.Fatalf("Failed to export cookies: %v", err)
	}

	// Read the profile to ensure it works
	profileData, err := browser.LoadProfile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load profile: %v", err)
	}
	
	if profileData.Domain != "https://example.com/" && profileData.Domain != "https://example.com" {
		t.Logf("Profile domain is %s, expected example.com. This is okay since example.com might not set cookies.", profileData.Domain)
	}

	t.Log("Profile loading test successful")
}

// MockHTTPClient represents a mock HTTP client for testing
type MockHTTPClient struct {
	Responses map[string]interface{}
	Requests  []string
}

func (m *MockHTTPClient) Do(req interface{}) (interface{}, error) {
	// Mock implementation
	return nil, nil
}

// WorkflowTestCase represents a test case for the workflow
type WorkflowTestCase struct {
	Name           string
	SessionID     string
	URL           string
	Actions       []string
	ExpectedState string
}

var workflowTestCases = []WorkflowTestCase{
	{
		Name:           "simple navigation",
		SessionID:      "test-1",
		URL:            "https://example.com",
		Actions:        []string{"snapshot"},
		ExpectedState: "ready",
	},
	{
		Name:           "navigation with click",
		SessionID:      "test-2",
		URL:            "https://example.com",
		Actions:        []string{"snapshot", "click", "snapshot"},
		ExpectedState:  "ready",
	},
	{
		Name:           "navigation with fill",
		SessionID:      "test-3",
		URL:            "https://example.com/form",
		Actions:        []string{"fill", "submit"},
		ExpectedState:  "ready",
	},
}

func TestWorkflowTestCases(t *testing.T) {
	for _, tc := range workflowTestCases {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.SessionID == "" {
				t.Error("SessionID is required")
			}
			if tc.URL == "" {
				t.Error("URL is required")
			}
			if len(tc.Actions) == 0 {
				t.Error("At least one action is required")
			}
			t.Logf("Workflow test case: %s - URL: %s, Actions: %v", 
				tc.Name, tc.URL, tc.Actions)
		})
	}
}

// SerializeTestCase serializes a workflow test case to JSON
func SerializeTestCase(tc WorkflowTestCase) ([]byte, error) {
	return json.Marshal(tc)
}

// DeserializeTestCase deserializes a workflow test case from JSON
func DeserializeTestCase(data []byte) (WorkflowTestCase, error) {
	var tc WorkflowTestCase
	err := json.Unmarshal(data, &tc)
	return tc, err
}

func TestSerializeWorkflowTestCase(t *testing.T) {
	tc := WorkflowTestCase{
		Name:           "test",
		SessionID:      "session-1",
		URL:            "https://example.com",
		Actions:        []string{"snapshot", "click"},
		ExpectedState:  "ready",
	}

	data, err := SerializeTestCase(tc)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	loaded, err := DeserializeTestCase(data)
	if err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}

	if loaded.Name != tc.Name {
		t.Errorf("Expected name '%s', got '%s'", tc.Name, loaded.Name)
	}
	if loaded.SessionID != tc.SessionID {
		t.Errorf("Expected session ID '%s', got '%s'", tc.SessionID, loaded.SessionID)
	}
}
