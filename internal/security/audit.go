package security

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// AuditEntry represents an audit log entry
type AuditEntry struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time             `json:"timestamp"`
	SessionID   string                 `json:"session_id"`
	AgentID     string                 `json:"agent_id,omitempty"`
	Action      string                 `json:"action"`
	TargetRef   string                 `json:"target_ref,omitempty"`
	TargetIntent string                `json:"target_intent,omitempty"`
	Domain     string                 `json:"domain,omitempty"`
	Reversibility string               `json:"reversibility"`
	ConfirmedBy string                 `json:"confirmed_by,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Result     string                 `json:"result"`
	Warnings   []Warning              `json:"warnings,omitempty"`
	PrevHash   string                 `json:"prev_hash"`
	ThisHash   string                 `json:"this_hash"`
}

// Warning represents a security warning
type Warning struct {
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// AuditLogger provides audit logging
type AuditLogger struct {
	lastHash string
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger() *AuditLogger {
	return &AuditLogger{
		lastHash: "genesis",
	}
}

// LogAction logs an action to the audit trail
func (al *AuditLogger) LogAction(entry *AuditEntry) error {
	entry.Timestamp = time.Now().UTC()
	entry.PrevHash = al.lastHash

	// Generate hash
	hashInput := fmt.Sprintf("%s|%s|%s|%s|%s",
		entry.Timestamp.Format(time.RFC3339Nano),
		entry.SessionID,
		entry.Action,
		entry.TargetRef,
		entry.PrevHash,
	)
	hash := sha256.Sum256([]byte(hashInput))
	entry.ThisHash = hex.EncodeToString(hash[:])

	// Update last hash
	al.lastHash = entry.ThisHash

	return nil
}

// VerifyChain verifies the audit log chain
func (al *AuditLogger) VerifyChain(entries []AuditEntry) bool {
	expectedPrev := "genesis"

	for _, entry := range entries {
		if entry.PrevHash != expectedPrev {
			return false
		}
		expectedPrev = entry.ThisHash
	}

	return true
}

// GetLastHash returns the last hash
func (al *AuditLogger) GetLastHash() string {
	return al.lastHash
}

// MarshalJSON marshals audit entry to JSON
func (al *AuditLogger) MarshalJSON(entry *AuditEntry) ([]byte, error) {
	type Alias AuditEntry
	return json.Marshal(&struct {
		*Alias
		Timestamp string `json:"timestamp"`
	}{
		Alias:     (*Alias)(entry),
		Timestamp: entry.Timestamp.Format(time.RFC3339),
	})
}
