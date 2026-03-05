package browser

import (
	"crypto/rand"
	"testing"

	"github.com/rennaisance-jomt/axon/internal/security"
	"github.com/rennaisance-jomt/axon/internal/storage"
)

func TestSuggestVaultSecrets(t *testing.T) {
	dbPath := t.TempDir()
	db, err := storage.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer db.Close()

	key := make([]byte, 32)
	rand.Read(key)
	vault := security.NewVault(db, key)

	// Add a secret
	err = vault.AddSecret(&security.Secret{
		Name:     "corp-admin",
		Domain:   "example.com",
		Username: "admin@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Failed to add secret: %v", err)
	}

	extractor := NewSnapshotExtractor().WithVault(vault)

	tests := []struct {
		name     string
		elements []Element
		check    func(t *testing.T, elements []Element)
	}{
		{
			name: "Email textbox (no keyword but has @ in placeholder)",
			elements: []Element{
				{Type: "textbox", Label: "Email Address", Placeholder: "example@organization.com"},
			},
			check: func(t *testing.T, elements []Element) {
				expected := "@vault:corp-admin:username"
				if elements[0].VaultSuggestion != expected {
					t.Errorf("Expected %q, got %q", expected, elements[0].VaultSuggestion)
				}
			},
		},
		{
			name: "HTML type email",
			elements: []Element{
				{Type: "email", Label: "Username", Placeholder: "Enter username"},
			},
			check: func(t *testing.T, elements []Element) {
				expected := "@vault:corp-admin:username"
				if elements[0].VaultSuggestion != expected {
					t.Errorf("Expected %q, got %q", expected, elements[0].VaultSuggestion)
				}
			},
		},
		{
			name: "Password field",
			elements: []Element{
				{Type: "password", Label: "Password", Placeholder: "Enter password"},
			},
			check: func(t *testing.T, elements []Element) {
				expected := "@vault:corp-admin:password"
				if elements[0].VaultSuggestion != expected {
					t.Errorf("Expected %q, got %q", expected, elements[0].VaultSuggestion)
				}
			},
		},
		{
			name: "input_group should be ignored for vault suggestion",
			elements: []Element{
				{Type: "input_group", Label: "Password / Login", Placeholder: ""},
			},
			check: func(t *testing.T, elements []Element) {
				if elements[0].VaultSuggestion != "" {
					t.Errorf("Expected empty suggestion for input_group, got %q", elements[0].VaultSuggestion)
				}
			},
		},
		{
			name: "Regular input with login keyword",
			elements: []Element{
				{Type: "textbox", Label: "Login id", Placeholder: ""},
			},
			check: func(t *testing.T, elements []Element) {
				expected := "@vault:corp-admin:username"
				if elements[0].VaultSuggestion != expected {
					t.Errorf("Expected %q, got %q", expected, elements[0].VaultSuggestion)
				}
			},
		},
		{
			name: "Text area should be ignored",
			elements: []Element{
				{Type: "textarea", Label: "Comments", Placeholder: "Enter test"},
			},
			check: func(t *testing.T, elements []Element) {
				if elements[0].VaultSuggestion != "" {
					t.Errorf("Expected empty suggestion for textarea, got %q", elements[0].VaultSuggestion)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh copy to avoid cross-test mutation
			elements := make([]Element, len(tt.elements))
			copy(elements, tt.elements)

			extractor.suggestVaultSecrets("https://example.com/login", elements)

			tt.check(t, elements)
		})
	}
}
