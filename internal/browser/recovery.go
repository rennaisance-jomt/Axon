package browser

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"
)

// FailureType represents types of failures that can occur
type FailureType string

const (
	FailureTypeNone           FailureType = "none"
	FailureTypeAccessDenied   FailureType = "access_denied"
	FailureTypeInvalidInput   FailureType = "invalid_input"
	FailureTypeTimeout        FailureType = "timeout"
	FailureTypeNetworkError   FailureType = "network_error"
	FailureTypeElementNotFound FailureType = "element_not_found"
	FailureTypeCaptcha        FailureType = "captcha"
	FailureTypeUnknown        FailureType = "unknown"
)

// FailurePattern represents a pattern to detect a specific failure
type FailurePattern struct {
	Type       FailureType
	Patterns   []string
	Regex      *regexp.Regexp
	Retryable  bool
	Severity   int // 1-10, higher is more severe
}

// ActionResult represents the result of an action
type ActionResult struct {
	Success      bool
	Error        error
	FailureType  FailureType
	RetriesUsed  int
	Timestamp    time.Time
	Description  string
}

// RecoveryConfig holds configuration for autonomous recovery
type RecoveryConfig struct {
	MaxRetries           int
	MaxRetriesPerAction  int
	BackoffBase         time.Duration
	BackoffMultiplier   float64
	MaxBackoff          time.Duration
	EnableAutoRollback  bool
	FailurePatterns     []FailurePattern
}

// DefaultRecoveryConfig returns default recovery configuration
func DefaultRecoveryConfig() *RecoveryConfig {
	return &RecoveryConfig{
		MaxRetries:          3,
		MaxRetriesPerAction: 3,
		BackoffBase:         500 * time.Millisecond,
		BackoffMultiplier:   2.0,
		MaxBackoff:         10 * time.Second,
		EnableAutoRollback: true,
		FailurePatterns:    defaultFailurePatterns(),
	}
}

// defaultFailurePatterns returns the default failure detection patterns
func defaultFailurePatterns() []FailurePattern {
	return []FailurePattern{
		{
			Type:      FailureTypeAccessDenied,
			Patterns:  []string{"access denied", "access denied", "403", "forbidden", "unauthorized"},
			Regex:     regexp.MustCompile(`(?i)(access denied|403|forbidden|unauthorized)`),
			Retryable: false,
			Severity:  8,
		},
		{
			Type:      FailureTypeInvalidInput,
			Patterns:  []string{"invalid input", "invalid email", "invalid password", "validation error"},
			Regex:     regexp.MustCompile(`(?i)(invalid (input|email|password)|validation error|incorrect)`),
			Retryable: true,
			Severity:  5,
		},
		{
			Type:      FailureTypeTimeout,
			Patterns:  []string{"timeout", "timed out", "taking too long"},
			Regex:     regexp.MustCompile(`(?i)(timeout|timed? ?out)`),
			Retryable: true,
			Severity:  3,
		},
		{
			Type:      FailureTypeNetworkError,
			Patterns:  []string{"network error", "connection failed", "connection refused", "dns error"},
			Regex:     regexp.MustCompile(`(?i)(network error|connection (failed|refused)|dns error)`),
			Retryable: true,
			Severity:  4,
		},
		{
			Type:      FailureTypeCaptcha,
			Patterns:  []string{"captcha", "recaptcha", "hcaptcha", "verify you are human"},
			Regex:     regexp.MustCompile(`(?i)(captcha|recaptcha|hcaptcha|verify (you are )?human)`),
			Retryable: false,
			Severity:  9,
		},
		{
			Type:      FailureTypeElementNotFound,
			Patterns:  []string{"element not found", "no such element", "cannot find"},
			Regex:     regexp.MustCompile(`(?i)(element not found|no such element|cannot find)`),
			Retryable: true,
			Severity:  3,
		},
	}
}

// RecoveryManager manages autonomous recovery for sessions
type RecoveryManager struct {
	mu           sync.RWMutex
	config       *RecoveryConfig
	checkpointMgr *CheckpointManager
	actionHistory map[string][]*ActionResult // sessionID -> action history
}

// NewRecoveryManager creates a new recovery manager
func NewRecoveryManager(config *RecoveryConfig, checkpointMgr *CheckpointManager) *RecoveryManager {
	if config == nil {
		config = DefaultRecoveryConfig()
	}
	return &RecoveryManager{
		config:        config,
		checkpointMgr: checkpointMgr,
		actionHistory: make(map[string][]*ActionResult),
	}
}

// DetectFailureType analyzes an error and determines the failure type
func (rm *RecoveryManager) DetectFailureType(err error) FailureType {
	if err == nil {
		return FailureTypeNone
	}

	errStr := err.Error()

	for _, pattern := range rm.config.FailurePatterns {
		if pattern.Regex != nil && pattern.Regex.MatchString(errStr) {
			return pattern.Type
		}
		for _, p := range pattern.Patterns {
			if strings.Contains(strings.ToLower(errStr), strings.ToLower(p)) {
				return pattern.Type
			}
		}
	}

	return FailureTypeUnknown
}

// IsRetryable determines if a failure type is retryable
func (rm *RecoveryManager) IsRetryable(failureType FailureType) bool {
	for _, pattern := range rm.config.FailurePatterns {
		if pattern.Type == failureType {
			return pattern.Retryable
		}
	}
	return false
}

// CalculateBackoff calculates the backoff duration for a retry
func (rm *RecoveryManager) CalculateBackoff(retriesUsed int) time.Duration {
	backoff := rm.config.BackoffBase
	for i := 0; i < retriesUsed; i++ {
		backoff = time.Duration(float64(backoff) * rm.config.BackoffMultiplier)
		if backoff > rm.config.MaxBackoff {
			backoff = rm.config.MaxBackoff
			break
		}
	}
	return backoff
}

// RecordAction records an action result for a session
func (rm *RecoveryManager) RecordAction(sessionID string, result *ActionResult) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	history, exists := rm.actionHistory[sessionID]
	if !exists {
		history = make([]*ActionResult, 0)
	}

	history = append(history, result)

	// Keep only last 100 actions
	if len(history) > 100 {
		history = history[len(history)-100:]
	}

	rm.actionHistory[sessionID] = history
}

// GetActionHistory returns the action history for a session
func (rm *RecoveryManager) GetActionHistory(sessionID string) []*ActionResult {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	history, exists := rm.actionHistory[sessionID]
	if !exists {
		return []*ActionResult{}
	}

	result := make([]*ActionResult, len(history))
	copy(result, history)
	return result
}

// GetFailureCount returns the number of failures for a session
func (rm *RecoveryManager) GetFailureCount(sessionID string) int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	history, exists := rm.actionHistory[sessionID]
	if !exists {
		return 0
	}

	count := 0
	for _, result := range history {
		if !result.Success {
			count++
		}
	}
	return count
}

// ShouldRollback determines if a session should be rolled back based on failure patterns
func (rm *RecoveryManager) ShouldRollback(sessionID string) (bool, *Checkpoint) {
	if !rm.config.EnableAutoRollback {
		return false, nil
	}

	failureCount := rm.GetFailureCount(sessionID)
	
	// If too many consecutive failures, suggest rollback
	if failureCount >= rm.config.MaxRetries {
		// Try to get the last checkpoint
		cp, err := rm.checkpointMgr.GetLatestCheckpoint(sessionID)
		if err != nil {
			log.Printf("Recovery: No checkpoint found for session %s", sessionID)
			return false, nil
		}
		return true, cp
	}

	return false, nil
}

// SuggestAlternativePath suggests an alternative approach when an action fails
func (rm *RecoveryManager) SuggestAlternativePath(action string, failureType FailureType) string {
	suggestions := map[FailureType]map[string]string{
		FailureTypeAccessDenied: {
			"click":   "Try clicking via JavaScript or using keyboard navigation",
			"fill":    "Try using JavaScript to set the value directly",
			"navigate": "Try a different URL or check if authentication is required",
		},
		FailureTypeInvalidInput: {
			"fill":    "Clear the field first, then fill. Check for input validation.",
			"click":   "Element may be disabled. Try waiting for it to become enabled.",
		},
		FailureTypeTimeout: {
			"click":   "Try using a more specific selector or waiting longer",
			"fill":    "Wait for the input to be ready before filling",
			"navigate": "Check network connectivity or try a simpler page",
		},
		FailureTypeElementNotFound: {
			"click":   "Element may have changed. Try using semantic search or take a new snapshot",
			"fill":    "Element may have changed. Try finding by label or placeholder",
		},
		FailureTypeCaptcha: {
			"navigate": "CAPTCHA detected. Manual intervention may be required.",
			"click":    "CAPTCHA detected. Cannot proceed automatically.",
		},
	}

	if actionSuggestions, ok := suggestions[failureType]; ok {
		if suggestion, ok := actionSuggestions[action]; ok {
			return suggestion
		}
	}

	return "Try taking a new snapshot and resolving elements again"
}

// CreatePreActionCheckpoint creates a checkpoint before a potentially irreversible action
func (rm *RecoveryManager) CreatePreActionCheckpoint(session *Session, actionDescription string) (*Checkpoint, error) {
	if rm.checkpointMgr == nil {
		return nil, fmt.Errorf("checkpoint manager not configured")
	}
	return rm.checkpointMgr.CreateCheckpoint(session, actionDescription)
}

// RollbackToCheckpoint rolls back a session to a specific checkpoint
func (rm *RecoveryManager) RollbackToCheckpoint(session *Session, checkpoint *Checkpoint) error {
	if rm.checkpointMgr == nil {
		return fmt.Errorf("checkpoint manager not configured")
	}
	log.Printf("Recovery: Rolling back session %s to checkpoint %s", session.ID, checkpoint.ID)
	return rm.checkpointMgr.RestoreFromCheckpoint(session, checkpoint)
}

// ClearHistory clears the action history for a session
func (rm *RecoveryManager) ClearHistory(sessionID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.actionHistory, sessionID)
}

// GetRecoveryStats returns recovery statistics for a session
func (rm *RecoveryManager) GetRecoveryStats(sessionID string) map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	history, exists := rm.actionHistory[sessionID]
	if !exists {
		return map[string]interface{}{
			"total_actions":   0,
			"successful":     0,
			"failed":         0,
			"retryable":      0,
			"non_retryable":  0,
		}
	}

	total := len(history)
	successful := 0
	retryable := 0
	nonRetryable := 0

	for _, result := range history {
		if result.Success {
			successful++
		} else if rm.IsRetryable(result.FailureType) {
			retryable++
		} else {
			nonRetryable++
		}
	}

	return map[string]interface{}{
		"total_actions":   total,
		"successful":     successful,
		"failed":         total - successful,
		"retryable":      retryable,
		"non_retryable":  nonRetryable,
	}
}
