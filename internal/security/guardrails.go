package security

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"sync"
	"time"
)

// GuardrailCategory represents the type of guardrail
type GuardrailCategory string

const (
	// CategoryViolence checks for violent content
	CategoryViolence GuardrailCategory = "violence"
	// CategorySexual checks for sexual content
	CategorySexual GuardrailCategory = "sexual"
	// CategoryHate checks for hate speech
	CategoryHate GuardrailCategory = "hate"
	// CategorySelfHarm checks for self-harm content
	CategorySelfHarm GuardrailCategory = "self_harm"
	// CategoryPII checks for personally identifiable information
	CategoryPII GuardrailCategory = "pii"
	// CategoryPromptInjection checks for prompt injection
	CategoryPromptInjection GuardrailCategory = "prompt_injection"
)

// GuardrailResult represents the result of a guardrail check
type GuardrailResult struct {
	Category    GuardrailCategory `json:"category"`
	Allowed     bool              `json:"allowed"`
	Confidence  float64           `json:"confidence"`
	MatchedText string            `json:"matched_text,omitempty"`
	Action      string            `json:"action"` // "block", "warn", "allow"
}

// GuardrailConfig holds guardrail configuration
type GuardrailConfig struct {
	Enabled       bool
	Categories    []GuardrailCategory
	Threshold     float64
	LLMEndpoint   string
	LLMAPIKey     string
	LocalModel    string
	UseLocalModel bool
}

// GuardrailManager manages content guardrails
type GuardrailManager struct {
	mu       sync.RWMutex
	config   *GuardrailConfig
	rules    map[GuardrailCategory][]*GuardrailRule
	client   *LLMClient
}

// GuardrailRule represents a single guardrail rule
type GuardrailRule struct {
	ID          string
	Category    GuardrailCategory
	Pattern     string
	Regex       *regexp.Regexp
	Weight      float64
	Description string
}

// LLMClient represents a local LLM client
type LLMClient struct {
	endpoint   string
	apiKey     string
	model      string
	mu         sync.RWMutex
	lastUsed   time.Time
}

// NewGuardrailManager creates a new guardrail manager
func NewGuardrailManager(cfg *GuardrailConfig) (*GuardrailManager, error) {
	if cfg == nil {
		cfg = &GuardrailConfig{
			Enabled:    true,
			Threshold:  0.8,
			Categories: []GuardrailCategory{CategoryViolence, CategorySexual, CategoryHate, CategoryPromptInjection},
		}
	}

	gm := &GuardrailManager{
		config: cfg,
		rules:  make(map[GuardrailCategory][]*GuardrailRule),
	}

	// Initialize default rules
	gm.initDefaultRules()

	// Initialize LLM client if configured
	if cfg.UseLocalModel && cfg.LLMEndpoint != "" {
		gm.client = &LLMClient{
			endpoint: cfg.LLMEndpoint,
			apiKey:   cfg.LLMAPIKey,
			model:    cfg.LocalModel,
		}
	}

	return gm, nil
}

func (gm *GuardrailManager) initDefaultRules() {
	// Violence patterns
	gm.rules[CategoryViolence] = []*GuardrailRule{
		{ID: "v1", Category: CategoryViolence, Pattern: `(?i)(kill|murder|attack|harm|shoot|stab)`, Weight: 0.7, Description: "Violent action"},
		{ID: "v2", Category: CategoryViolence, Pattern: `(?i)(weapon|gun|knife|bomb)`, Weight: 0.6, Description: "Weapon reference"},
	}

	// Sexual patterns
	gm.rules[CategorySexual] = []*GuardrailRule{
		{ID: "s1", Category: CategorySexual, Pattern: `(?i)(nude|naked|explicit)`, Weight: 0.8, Description: "Sexual content"},
	}

	// Hate speech patterns
	gm.rules[CategoryHate] = []*GuardrailRule{
		{ID: "h1", Category: CategoryHate, Pattern: `(?i)(hate|slur|discriminate)`, Weight: 0.9, Description: "Hate speech"},
	}

	// Prompt injection patterns
	gm.rules[CategoryPromptInjection] = []*GuardrailRule{
		{ID: "pi1", Category: CategoryPromptInjection, Pattern: `(?i)(ignore (previous|all)|forget (your|this)|system( prompt)?:|\[INST\])`, Weight: 0.9, Description: "Prompt injection"},
		{ID: "pi2", Category: CategoryPromptInjection, Pattern: `(?i)(jailbreak|hack|override)`, Weight: 0.7, Description: "Jailbreak attempt"},
		{ID: "pi3", Category: CategoryPromptInjection, Pattern: `<\/?(script|iframe|style)`, Weight: 0.8, Description: "HTML injection"},
	}

	// PII patterns
	gm.rules[CategoryPII] = []*GuardrailRule{
		{ID: "p1", Category: CategoryPII, Pattern: `\b\d{3}-\d{2}-\d{4}\b`, Weight: 0.9, Description: "SSN"},
		{ID: "p2", Category: CategoryPII, Pattern: `\b\d{16}\b`, Weight: 0.8, Description: "Credit card"},
		{ID: "p3", Category: CategoryPII, Pattern: `(?i)\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b`, Weight: 0.7, Description: "Email"},
	}

	// Compile regex patterns
	for category, rules := range gm.rules {
		for _, rule := range rules {
			re, err := regexp.Compile(rule.Pattern)
			if err != nil {
				continue
			}
			rule.Regex = re
		}
		gm.rules[category] = rules
	}
}

// CheckContent checks content against guardrails
func (gm *GuardrailManager) CheckContent(ctx context.Context, content string) ([]*GuardrailResult, error) {
	if !gm.config.Enabled {
		return []*GuardrailResult{}, nil
	}

	results := make([]*GuardrailResult, 0)

	// Run regex-based checks
	for _, category := range gm.config.Categories {
		rules, ok := gm.rules[category]
		if !ok {
			continue
		}

		for _, rule := range rules {
			matches := rule.Regex.FindStringSubmatch(content)
			if len(matches) > 0 {
				result := &GuardrailResult{
					Category:    category,
					Allowed:     false,
					Confidence:  rule.Weight,
					MatchedText: matches[0],
					Action:      "block",
				}
				results = append(results, result)
			}
		}
	}

	// Run LLM-based check if enabled and regex didn't find clear violations
	if gm.client != nil && len(results) == 0 {
		llmResults, err := gm.client.CheckWithLLM(ctx, content, gm.config.Categories)
		if err == nil {
			results = append(results, llmResults...)
		}
	}

	return results, nil
}

// CheckPrompt checks a prompt for injection attacks
func (gm *GuardrailManager) CheckPrompt(ctx context.Context, prompt string) (*GuardrailResult, error) {
	results, err := gm.CheckContent(ctx, prompt)
	if err != nil {
		return nil, err
	}

	for _, result := range results {
		if result.Category == CategoryPromptInjection && !result.Allowed {
			return result, nil
		}
	}

	return &GuardrailResult{
		Category:   CategoryPromptInjection,
		Allowed:    true,
		Confidence: 1.0,
		Action:     "allow",
	}, nil
}

// CheckPII checks for PII in content
func (gm *GuardrailManager) CheckPII(ctx context.Context, content string) ([]*GuardrailResult, error) {
	results, err := gm.CheckContent(ctx, content)
	if err != nil {
		return nil, err
	}

	piiResults := make([]*GuardrailResult, 0)
	for _, result := range results {
		if result.Category == CategoryPII && !result.Allowed {
			piiResults = append(piiResults, result)
		}
	}

	return piiResults, nil
}

// AddRule adds a custom guardrail rule
func (gm *GuardrailManager) AddRule(rule *GuardrailRule) error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	re, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return err
	}
	rule.Regex = re

	gm.rules[rule.Category] = append(gm.rules[rule.Category], rule)
	return nil
}

// CheckWithLLM checks content using LLM (stub)
func (c *LLMClient) CheckWithLLM(ctx context.Context, content string, categories []GuardrailCategory) ([]*GuardrailResult, error) {
	// This would call a local LLM like Llama-Guard
	// For now, return empty results
	return []*GuardrailResult{}, nil
}

// IsAllowed checks if content is allowed based on results
func (gm *GuardrailManager) IsAllowed(results []*GuardrailResult) bool {
	for _, result := range results {
		if !result.Allowed && result.Confidence >= gm.config.Threshold {
			return false
		}
	}
	return true
}

// GetBlockedCategories returns categories that should be blocked
func (gm *GuardrailManager) GetBlockedCategories(results []*GuardrailResult) []GuardrailCategory {
	blocked := make([]GuardrailCategory, 0)
	for _, result := range results {
		if !result.Allowed && result.Confidence >= gm.config.Threshold {
			blocked = append(blocked, result.Category)
		}
	}
	return blocked
}

// AnonymizeContent replaces PII with placeholders
func (gm *GuardrailManager) AnonymizeContent(content string) string {
	result := content

	// Replace SSN
	ssnRe := regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	result = ssnRe.ReplaceAllString(result, "[SSN]")

	// Replace credit cards
	ccRe := regexp.MustCompile(`\b\d{16}\b`)
	result = ccRe.ReplaceAllString(result, "[CC]")

	// Replace emails (simplified)
	emailRe := regexp.MustCompile(`(?i)\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b`)
	result = emailRe.ReplaceAllString(result, "[EMAIL]")

	// Replace phone numbers
	phoneRe := regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`)
	result = phoneRe.ReplaceAllString(result, "[PHONE]")

	return result
}

// ValidateURL checks if URL is safe to navigate
func (gm *GuardrailManager) ValidateURL(url string) error {
	// Check for malicious patterns
	maliciousPatterns := []string{
		"javascript:",
		"data:text/html",
		"vbscript:",
	}

	lowerURL := strings.ToLower(url)
	for _, pattern := range maliciousPatterns {
		if strings.Contains(lowerURL, pattern) {
			return &GuardrailError{
				Category: "url_validation",
				Message:  "URL contains potentially malicious pattern: " + pattern,
			}
		}
	}

	return nil
}

// GuardrailError represents a guardrail violation error
type GuardrailError struct {
	Category string
	Message  string
}

func (e *GuardrailError) Error() string {
	return e.Message
}

// MarshalJSON implements json.Marshaler
func (e *GuardrailError) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"category": e.Category,
		"message":  e.Message,
	})
}
