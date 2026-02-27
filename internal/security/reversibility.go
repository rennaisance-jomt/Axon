package security

import (
	"strings"
)

// Reversibility levels
const (
	ReversibilityRead             = "read"
	ReversibilityWriteReversible  = "write_reversible"
	ReversibilityWriteIrreversible = "write_irreversible"
	ReversibilitySensitiveWrite   = "sensitive_write"
)

// ActionClassifier classifies actions by reversibility
type ActionClassifier struct {
	irreversibleKeywords []string
	sensitiveKeywords   []string
}

// NewActionClassifier creates a new classifier
func NewActionClassifier() *ActionClassifier {
	return &ActionClassifier{
		irreversibleKeywords: []string{
			"delete", "remove", "destroy", "erase",
			"post", "publish", "submit", "send",
			"buy", "purchase", "order", "pay",
			"transfer", "wire", "withdraw",
			"close", "cancel", "unsubscribe",
			"update", "change", "modify",
		},
		sensitiveKeywords: []string{
			"password", "passwd", "secret",
			"credit", "card", "cvv", "ssn",
			"api", "key", "token", "auth",
		},
	}
}

// ClassifyAction classifies an action by reversibility
func (ac *ActionClassifier) ClassifyAction(actionType, elementLabel, elementType string) string {
	label := strings.ToLower(elementLabel)

	// Check for irreversible actions
	for _, keyword := range ac.irreversibleKeywords {
		if strings.Contains(label, keyword) {
			return ReversibilityWriteIrreversible
		}
	}

	// Check for sensitive write actions
	for _, keyword := range ac.sensitiveKeywords {
		if strings.Contains(label, keyword) || elementType == "password" {
			return ReversibilitySensitiveWrite
		}
	}

	// Classify by action type
	switch actionType {
	case "click":
		// Click on button/link is generally reversible
		if elementType == "button" || elementType == "a" || elementType == "link" {
			return ReversibilityWriteReversible
		}
	case "fill", "type":
		// Filling text is reversible (can be cleared)
		return ReversibilityWriteReversible
	case "select":
		// Selecting options is reversible
		return ReversibilityWriteReversible
	case "navigate", "snapshot", "screenshot":
		// Read-only operations
		return ReversibilityRead
	}

	return ReversibilityWriteReversible
}

// RequiresConfirmation checks if an action requires explicit confirmation
func (ac *ActionClassifier) RequiresConfirmation(reversibility string) bool {
	return reversibility == ReversibilityWriteIrreversible || reversibility == ReversibilitySensitiveWrite
}

// IsSafe checks if an action is considered safe
func (ac *ActionClassifier) IsSafe(reversibility string) bool {
	return reversibility == ReversibilityRead || reversibility == ReversibilityWriteReversible
}
