package storage

import (
	"bytes"
	"os"
	"testing"
)

func TestStorageOperations(t *testing.T) {
	// Create a temporary directory for Badger
	dir, err := os.MkdirTemp("", "badger-test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory for badger: %v", err)
	}
	defer os.RemoveAll(dir) // Cleanup after test

	// Open DB
	db, err := New(dir)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	// --- Session Operations ---
	sessionID := "test-session"
	sessionData := []byte(`{"id": "test-session", "status": "active"}`)

	// Set session
	if err := db.SetSession(sessionID, sessionData); err != nil {
		t.Fatalf("SetSession failed: %v", err)
	}

	// Get session
	retrievedData, err := db.GetSession(sessionID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if !bytes.Equal(retrievedData, sessionData) {
		t.Fatalf("Expected session data %s, got %s", sessionData, retrievedData)
	}

	// List sessions
	sessions, err := db.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(sessions) != 1 || sessions[0] != sessionID {
		t.Fatalf("Expected 1 session with ID %s, got %v", sessionID, sessions)
	}

	// Delete session
	if err := db.DeleteSession(sessionID); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	// Trying to get deleted session should fail
	_, err = db.GetSession(sessionID)
	if err == nil {
		t.Fatalf("Expected error when getting deleted session")
	}

	// --- Audit Logs ---
	log1 := []byte(`{"action": "navigate"}`)
	log2 := []byte(`{"action": "click"}`)

	if err := db.AppendAuditLog(log1); err != nil {
		t.Fatalf("AppendAuditLog failed: %v", err)
	}
	if err := db.AppendAuditLog(log2); err != nil {
		t.Fatalf("AppendAuditLog failed: %v", err)
	}

	logs, err := db.GetAuditLogs(10, 0)
	if err != nil {
		t.Fatalf("GetAuditLogs failed: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("Expected 2 logs, got %d", len(logs))
	}
	if !bytes.Equal(logs[0], log1) || !bytes.Equal(logs[1], log2) {
		t.Fatalf("Logs retrieved mismatch. Expected %s and %s, but got %s and %s", log1, log2, logs[0], logs[1])
	}
	
	// Test limit and offset
	offsetLogs, err := db.GetAuditLogs(1, 1)
	if err != nil {
		t.Fatalf("GetAuditLogs failed: %v", err)
	}
	if len(offsetLogs) != 1 || !bytes.Equal(offsetLogs[0], log2) {
		t.Fatalf("Offset logs mismatch. Expected %s, got %v", log2, offsetLogs)
	}

	// --- Element Memory ---
	domain := "example.com"
	memoryData := []byte(`{"btn-login": "button[type='submit']"}`)

	if err := db.SetElementMemory(domain, memoryData); err != nil {
		t.Fatalf("SetElementMemory failed: %v", err)
	}

	retrievedMemory, err := db.GetElementMemory(domain)
	if err != nil {
		t.Fatalf("GetElementMemory failed: %v", err)
	}
	if !bytes.Equal(retrievedMemory, memoryData) {
		t.Fatalf("Expected memory %s, got %s", memoryData, retrievedMemory)
	}
}
