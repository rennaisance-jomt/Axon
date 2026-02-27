package browser

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-rod/rod/lib/proto"
)

// Profile represents a session profile
type Profile struct {
	Name         string         `json:"name"`
	Domain       string         `json:"domain"`
	Cookies      []*proto.NetworkCookie `json:"cookies"`
	LocalStorage map[string]string `json:"local_storage,omitempty"`
	CreatedAt    string         `json:"created_at"`
	LastUsed     string         `json:"last_used,omitempty"`
}

// LoadProfile loads a profile from a file
func LoadProfile(path string) (*Profile, error) {
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile: %w", err)
	}

	var profile Profile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("failed to parse profile: %w", err)
	}

	return &profile, nil
}

// SaveProfile saves a profile to a file
func SaveProfile(path string, profile *Profile) error {
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal profile: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write profile: %w", err)
	}

	return nil
}

// ExportCookies exports cookies from a session to a file
func (s *Session) ExportCookies(path string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cookies, err := s.Page.Cookies()
	if err != nil {
		return fmt.Errorf("failed to get cookies: %w", err)
	}

	profile := &Profile{
		Domain:  s.URL,
		Cookies: cookies,
	}

	return SaveProfile(path, profile)
}

// ImportCookies imports cookies from a file into a session
func (s *Session) ImportCookies(path string) error {
	profile, err := LoadProfile(path)
	if err != nil {
		return err
	}

	if profile == nil || len(profile.Cookies) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Convert to NetworkCookieParam
	var cookieParams []*proto.NetworkCookieParam
	for _, c := range profile.Cookies {
		cookieParams = append(cookieParams, &proto.NetworkCookieParam{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  c.Expires,
			HTTPOnly: c.HTTPOnly,
			Secure:   c.Secure,
			SameSite: c.SameSite,
		})
	}

	return s.Page.SetCookies(cookieParams)
}

// GetCookies gets all cookies from a session
func (s *Session) GetCookies(domain string) ([]*proto.NetworkCookie, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cookies, err := s.Page.Cookies()
	if err != nil {
		return nil, err
	}

	// Filter by domain if specified
	if domain != "" {
		var filtered []*proto.NetworkCookie
		for _, c := range cookies {
			if containsDomain(c.Domain, domain) {
				filtered = append(filtered, c)
			}
		}
		return filtered, nil
	}

	return cookies, nil
}

// SetCookies sets cookies in a session
func (s *Session) SetCookies(cookies []*proto.NetworkCookieParam) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.Page.SetCookies(cookies)
}

// ClearCookies clears all cookies in a session
func (s *Session) ClearCookies() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get all cookies
	cookies, err := s.Page.Cookies()
	if err != nil {
		return err
	}

	// Delete each cookie
	for _, c := range cookies {
		if err := s.Page.DeleteCookie(c.Name); err != nil {
			return err
		}
	}

	return nil
}

func containsDomain(cookieDomain, target string) bool {
	cookieDomain = removeLeadingDot(cookieDomain)
	return cookieDomain == target || 
		   cookieDomain == "" || 
		   target == "" ||
		   hasSuffix(cookieDomain, "."+target)
}

func removeLeadingDot(s string) string {
	if len(s) > 0 && s[0] == '.' {
		return s[1:]
	}
	return s
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
