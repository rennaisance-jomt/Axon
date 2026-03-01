package security

import (
	"strings"
)

// PromptInjectionGuard detects suspected prompt injection in page content
type PromptInjectionGuard struct {
	patterns []string
}

// NewPromptInjectionGuard creates a new prompt injection guard
func NewPromptInjectionGuard() *PromptInjectionGuard {
	return &PromptInjectionGuard{
		patterns: []string{
			"ignore all previous instructions",
			"ignore previous instructions",
			"system prompt",
			"you are now an ai that",
			"new instructions:",
			"override:",
			"assistant:",
			"user:",
		},
	}
}

// ScanContent scans text for suspected prompt injection
func (g *PromptInjectionGuard) ScanContent(content string) (bool, string) {
	lowerContent := strings.ToLower(content)
	for _, pattern := range g.patterns {
		if strings.Contains(lowerContent, pattern) {
			return true, pattern
		}
	}
	return false, ""
}
