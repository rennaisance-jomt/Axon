package browser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestLoadProfile(t *testing.T) {
	t.Run("load profile with empty path", func(t *testing.T) {
		profile, err := LoadProfile("")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if profile != nil {
			t.Error("Expected nil profile for empty path")
		}
	})

	t.Run("load non-existent profile", func(t *testing.T) {
		_, err := LoadProfile("/non/existent/path.json")
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})

	t.Run("load valid profile", func(t *testing.T) {
		// Create a temporary profile file
		profile := Profile{
			Name:    "test-profile",
			Domain:  "example.com",
			Cookies: nil,
			LocalStorage: map[string]string{
				"key1": "value1",
			},
			CreatedAt: "2024-01-01T00:00:00Z",
			LastUsed:  "2024-01-02T00:00:00Z",
		}

		data, err := json.Marshal(profile)
		if err != nil {
			t.Fatalf("Failed to marshal profile: %v", err)
		}

		tmpFile, err := os.CreateTemp("", "profile-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.Write(data); err != nil {
			t.Fatalf("Failed to write profile: %v", err)
		}
		tmpFile.Close()

		loaded, err := LoadProfile(tmpFile.Name())
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if loaded == nil {
			t.Fatal("Expected loaded profile, got nil")
		}
		if loaded.Name != "test-profile" {
			t.Errorf("Expected name 'test-profile', got '%s'", loaded.Name)
		}
		if loaded.Domain != "example.com" {
			t.Errorf("Expected domain 'example.com', got '%s'", loaded.Domain)
		}
		if len(loaded.LocalStorage) != 1 {
			t.Errorf("Expected 1 local storage entry, got %d", len(loaded.LocalStorage))
		}
	})

	t.Run("load invalid profile JSON", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "invalid-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.WriteString("invalid json"); err != nil {
			t.Fatalf("Failed to write: %v", err)
		}
		tmpFile.Close()

		_, err = LoadProfile(tmpFile.Name())
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})
}

func TestSaveProfile(t *testing.T) {
	t.Run("save profile", func(t *testing.T) {
		profile := &Profile{
			Name:    "save-test",
			Domain:  "test.com",
			Cookies: nil,
			LocalStorage: map[string]string{
				"token": "abc123",
			},
			CreatedAt: "2024-01-01T00:00:00Z",
		}

		tmpFile, err := os.CreateTemp("", "save-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		path := tmpFile.Name()
		tmpFile.Close()
		defer os.Remove(path)

		err = SaveProfile(path, profile)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify file was written
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		var loaded Profile
		if err := json.Unmarshal(data, &loaded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if loaded.Name != "save-test" {
			t.Errorf("Expected name 'save-test', got '%s'", loaded.Name)
		}
	})
}

func TestProfile_Structure(t *testing.T) {
	profile := Profile{
		Name:    "test",
		Domain:  "example.com",
		Cookies: nil,
		LocalStorage: map[string]string{
			"key": "value",
		},
		CreatedAt: "2024-01-01",
		LastUsed:  "2024-01-02",
	}

	if profile.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", profile.Name)
	}
	if profile.Domain != "example.com" {
		t.Errorf("Expected domain 'example.com', got '%s'", profile.Domain)
	}
	if profile.LocalStorage == nil {
		t.Error("Expected LocalStorage to be set")
	}
}

func TestContainsDomain(t *testing.T) {
	tests := []struct {
		name        string
		cookieDomain string
		target      string
		expected    bool
	}{
		{"exact match", "example.com", "example.com", true},
		{"with leading dot", ".example.com", "example.com", true},
		{"subdomain", "sub.example.com", "example.com", true},
		{"different domain", "other.com", "example.com", false},
		{"empty cookie domain", "", "example.com", true},
		{"empty target", "example.com", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsDomain(tt.cookieDomain, tt.target)
			if result != tt.expected {
				t.Errorf("containsDomain(%q, %q) = %v, expected %v", 
					tt.cookieDomain, tt.target, result, tt.expected)
			}
		})
	}
}

func TestRemoveLeadingDot(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{".example.com", "example.com"},
		{"example.com", "example.com"},
		{"", ""},
		{".", ""},
	}

	for _, tt := range tests {
		result := removeLeadingDot(tt.input)
		if result != tt.expected {
			t.Errorf("removeLeadingDot(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestHasSuffix(t *testing.T) {
	tests := []struct {
		input    string
		suffix   string
		expected bool
	}{
		{"example.com", ".com", true},
		{"example.com", "example", false},
		{"test", "st", true},
		{"test", "est", true},
		{"test", "test", true},
		{"test", "testing", false},
	}

	for _, tt := range tests {
		result := hasSuffix(tt.input, tt.suffix)
		if result != tt.expected {
			t.Errorf("hasSuffix(%q, %q) = %v, expected %v", 
				tt.input, tt.suffix, result, tt.expected)
		}
	}
}