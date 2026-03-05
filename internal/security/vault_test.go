package security

import (
	"crypto/rand"
	"testing"

	"github.com/rennaisance-jomt/axon/internal/storage"
)

func TestGetBaseDomain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Bare domain", "example.com", "example.com"},
		{"Subdomain", "login.example.com", "example.com"},
		{"Multi-level subdomain", "app.dev.example.com", "example.com"},
		{"HTTP URL", "http://example.com/login", "example.com"},
		{"HTTPS URL with sub", "https://auth.example.com/docs", "example.com"},
		{"Localhost", "localhost", "localhost"},
		// public suffix behavior on host with port acts on the string.
		// "localhost:8080" isn't a valid TLD so it usually returns it exactly as given.
		{"Localhost port", "localhost:8080", "localhost:8080"},
		{"URL Localhost port", "http://localhost:8080/login", "localhost"},
		{"Co.uk domain", "example.co.uk", "example.co.uk"},
		{"Co.uk subdomain", "login.example.co.uk", "example.co.uk"},
		{"IP address", "192.168.1.1", "192.168.1.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetBaseDomain(tt.input)
			if got != tt.expected {
				t.Errorf("GetBaseDomain(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestVault_AddAndGetSecret(t *testing.T) {
	dbPath := t.TempDir()
	db, err := storage.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer db.Close()

	key := make([]byte, 32)
	rand.Read(key)

	vault := NewVault(db, key)

	secret := &Secret{
		Name:     "test-admin",
		Domain:   "https://admin.example.com/login",
		Username: "admin",
		Password: "password123",
	}

	err = vault.AddSecret(secret)
	if err != nil {
		t.Fatalf("Failed to add secret: %v", err)
	}

	// Fetch from exact same domain URL
	fetched, err := vault.GetSecret("test-admin", "https://admin.example.com/login")
	if err != nil {
		t.Fatalf("Expected to find secret, got error: %v", err)
	}
	if fetched.Username != "admin" || fetched.Password != "password123" {
		t.Errorf("Fetched secret did not match: %+v", fetched)
	}

	// Fetch from bare base domain
	fetched2, err := vault.GetSecret("test-admin", "example.com")
	if err != nil {
		t.Fatalf("Expected to find secret by base domain, got error: %v", err)
	}
	if fetched2.Username != "admin" {
		t.Errorf("Fetched secret 2 did not match: %+v", fetched2)
	}

	// Fetch from completely different domain (should fail)
	_, err = vault.GetSecret("test-admin", "attacker.com")
	if err == nil {
		t.Fatal("Expected error fetching from attacker.com, got success")
	}

	// Test listing secrets
	secrets, err := vault.ListSecretsByDomain("example.com")
	if err != nil {
		t.Fatalf("ListSecretsByDomain failed: %v", err)
	}
	if len(secrets) != 1 {
		t.Errorf("Expected 1 secret, got %d", len(secrets))
	}
	if secrets[0].Name != "test-admin" {
		t.Errorf("Expected secret name 'test-admin', got %q", secrets[0].Name)
	}
}
