package mcp

import (
	"fmt"
	"strings"

	"github.com/rennaisance-jomt/axon/internal/browser"
	"github.com/rennaisance-jomt/axon/internal/storage"
)

// IntentResolver resolves natural language intents to element references
type IntentResolver struct {
	db *storage.DB
}

// NewIntentResolver creates a new intent resolver
func NewIntentResolver(db *storage.DB) *IntentResolver {
	return &IntentResolver{db: db}
}

// ElementMatch represents a potential element match
type ElementMatch struct {
	Ref      string
	Score    float64
	Reason   string
}

// Resolve finds the best matching element for a given intent
func (r *IntentResolver) Resolve(session *browser.Session, intent string) (string, error) {
	intent = strings.ToLower(strings.TrimSpace(intent))
	
	// Get current elements from session
	elements := session.GetLastElements()
	if len(elements) == 0 {
		return "", fmt.Errorf("no elements available in current snapshot")
	}
	
	// Score all elements
	var matches []ElementMatch
	for _, el := range elements {
		score, reason := r.scoreElement(&el, intent)
		if score > 0 {
			matches = append(matches, ElementMatch{
				Ref:    el.Ref,
				Score:  score,
				Reason: reason,
			})
		}
	}
	
	// Find best match
	if len(matches) == 0 {
		return "", fmt.Errorf("no element matches intent: %s", intent)
	}
	
	// Sort by score (simple bubble sort for now)
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].Score > matches[i].Score {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}
	
	bestMatch := matches[0]
	if bestMatch.Score < 0.3 {
		return "", fmt.Errorf("no confident match found for intent: %s (best score: %.2f)", intent, bestMatch.Score)
	}
	
	// Store successful resolution for learning
	r.storeResolution(session.URL, intent, bestMatch.Ref, bestMatch.Score)
	
	return bestMatch.Ref, nil
}

func (r *IntentResolver) scoreElement(el *browser.Element, intent string) (float64, string) {
	var score float64
	var reasons []string
	
	// Check label content match
	if el.Label != "" {
		textScore := r.calculateTextScore(el.Label, intent)
		if textScore > 0 {
			score += textScore * 0.4
			reasons = append(reasons, fmt.Sprintf("label match: %.2f", textScore))
		}
	}
	
	// Check placeholder match
	if el.Placeholder != "" {
		placeholderScore := r.calculateTextScore(el.Placeholder, intent)
		if placeholderScore > 0 {
			score += placeholderScore * 0.35
			reasons = append(reasons, fmt.Sprintf("placeholder match: %.2f", placeholderScore))
		}
	}
	
	// Check ARIA role match
	if el.Role != "" {
		roleScore := r.calculateRoleScore(el.Role, el.Type, intent)
		if roleScore > 0 {
			score += roleScore * 0.3
			reasons = append(reasons, fmt.Sprintf("role match: %.2f", roleScore))
		}
	}
	
	// Check element type match
	typeScore := r.calculateTypeScore(el.Type, intent)
	if typeScore > 0 {
		score += typeScore * 0.2
		reasons = append(reasons, fmt.Sprintf("type match: %.2f", typeScore))
	}
	
	// Check intent field
	if el.Intent != "" {
		intentScore := r.calculateTextScore(el.Intent, intent)
		if intentScore > 0 {
			score += intentScore * 0.15
			reasons = append(reasons, fmt.Sprintf("intent match: %.2f", intentScore))
		}
	}
	
	reason := strings.Join(reasons, ", ")
	return score, reason
}

func (r *IntentResolver) calculateTextScore(text, intent string) float64 {
	text = strings.ToLower(text)
	intentWords := strings.Fields(intent)
	
	// Direct containment
	if strings.Contains(text, intent) {
		return 1.0
	}
	
	// Check for keyword matches
	var matchCount float64
	for _, word := range intentWords {
		if len(word) < 3 {
			continue // Skip short words
		}
		if strings.Contains(text, word) {
			matchCount += 1.0
		}
	}
	
	if len(intentWords) > 0 {
		return matchCount / float64(len(intentWords))
	}
	
	return 0
}

func (r *IntentResolver) calculateRoleScore(role, elementType, intent string) float64 {
	role = strings.ToLower(role)
	intent = strings.ToLower(intent)
	
	// Role-specific intent mappings
	roleMappings := map[string][]string{
		"button":    {"click", "press", "submit", "button", "action"},
		"link":      {"navigate", "link", "go to", "visit", "open"},
		"textbox":   {"fill", "type", "enter", "input", "text", "search", "email", "password"},
		"searchbox": {"search", "find", "query", "look up"},
		"checkbox":  {"check", "select", "toggle", "checkbox"},
		"combobox":  {"select", "choose", "dropdown", "pick"},
	}
	
	// Check if role matches intent keywords
	keywords, exists := roleMappings[role]
	if !exists {
		keywords = roleMappings[elementType]
	}
	
	for _, keyword := range keywords {
		if strings.Contains(intent, keyword) {
			return 0.8
		}
	}
	
	return 0
}

func (r *IntentResolver) calculateTypeScore(elementType, intent string) float64 {
	elementType = strings.ToLower(elementType)
	intent = strings.ToLower(intent)
	
	// Type-specific intent mappings
	typeMappings := map[string][]string{
		"button":   {"button", "click", "submit", "action"},
		"input":    {"fill", "type", "enter", "input", "field"},
		"textarea": {"fill", "type", "enter", "text", "message", "description"},
		"select":   {"select", "choose", "dropdown", "pick"},
		"a":        {"link", "navigate", "go"},
	}
	
	keywords, exists := typeMappings[elementType]
	if !exists {
		return 0
	}
	
	for _, keyword := range keywords {
		if strings.Contains(intent, keyword) {
			return 0.6
		}
	}
	
	return 0
}

func (r *IntentResolver) storeResolution(url, intent, ref string, score float64) {
	// Only store high-confidence matches
	if score < 0.5 {
		return
	}
	
	// Extract domain from URL
	domain := extractDomain(url)
	if domain == "" {
		return
	}
	
	// Store in database for future use
	key := fmt.Sprintf("intent:%s:%s", domain, hashIntent(intent))
	value := fmt.Sprintf("%s|%.2f", ref, score)
	
	if r.db != nil {
		r.db.StoreElementMemory(key, value)
	}
}

// extractDomain extracts the domain from a URL
func extractDomain(url string) string {
	// Simple extraction - remove protocol and path
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	
	// Get just the domain part
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		domain := parts[0]
		// Remove port if present
		if idx := strings.Index(domain, ":"); idx != -1 {
			domain = domain[:idx]
		}
		return domain
	}
	return ""
}

// hashIntent creates a simple hash of the intent for storage
func hashIntent(intent string) string {
	// Simple hash - just lowercase and replace spaces
	return strings.ReplaceAll(strings.ToLower(intent), " ", "_")
}
