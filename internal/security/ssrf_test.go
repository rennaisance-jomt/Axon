package security

import (
	"testing"
)

func TestSSRFGuard_ValidateURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		allowlist []string
		denylist  []string
		wantErr   bool
	}{
		{
			name:    "valid HTTPS URL",
			url:     "https://example.com",
			wantErr: false,
		},
		{
			name:    "valid HTTP URL",
			url:     "http://example.com",
			wantErr: false,
		},
		{
			name:    "javascript scheme blocked",
			url:     "javascript:alert(1)",
			wantErr: true,
		},
		{
			name:    "data scheme blocked",
			url:     "data:text/html,<script>alert(1)</script>",
			wantErr: true,
		},
		{
			name:    "file scheme blocked",
			url:     "file:///etc/passwd",
			wantErr: true,
		},
		{
			name:    "private IP blocked",
			url:     "http://192.168.1.1",
			wantErr: true,
		},
		{
			name:    "localhost blocked",
			url:     "http://127.0.0.1",
			wantErr: true,
		},
		{
			name:    "link-local blocked",
			url:     "http://169.254.0.1",
			wantErr: true,
		},
		{
			name:      "allowlist - domain in allowlist",
			url:       "https://example.com",
			allowlist: []string{"example.com"},
			wantErr:   false,
		},
		{
			name:      "allowlist - domain not in allowlist",
			url:       "https://evil.com",
			allowlist: []string{"example.com"},
			wantErr:   true,
		},
		{
			name:     "denylist - domain in denylist",
			url:      "https://evil.com",
			denylist: []string{"evil.com"},
			wantErr:  true,
		},
		{
			name:    "invalid URL",
			url:     "://invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guard := NewSSRFGuard(false, tt.allowlist, tt.denylist, []string{"https", "http"})
			err := guard.ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSSRFGuard_AllowPrivateNetwork(t *testing.T) {
	// When allowPrivateNetwork is true, private URLs should work
	guard := NewSSRFGuard(true, []string{}, []string{}, []string{"https", "http"})

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "private IP allowed",
			url:     "http://192.168.1.1",
			wantErr: false,
		},
		{
			name:    "localhost allowed",
			url:     "http://127.0.0.1",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := guard.ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
