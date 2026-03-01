package browser

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-rod/rod/lib/proto"
)

// Checkpoint represents a session state snapshot
type Checkpoint struct {
	ID             string                   `json:"id"`
	SessionID      string                   `json:"session_id"`
	Timestamp      time.Time                `json:"timestamp"`
	URL            string                   `json:"url"`
	Title          string                   `json:"title"`
	Cookies        []*proto.NetworkCookie   `json:"cookies"`
	LocalStorage   map[string]string        `json:"local_storage"`
	SessionStorage map[string]string        `json:"session_storage"`
	ActionIndex    int                      `json:"action_index"` // Position in action sequence
	Description    string                   `json:"description"`
}

// CheckpointManager manages session checkpoints
type CheckpointManager struct {
	mu             sync.RWMutex
	checkpoints    map[string][]*Checkpoint // sessionID -> checkpoints
	maxCheckpoints int                      // Maximum checkpoints per session
	ttl            time.Duration            // Time to live for checkpoints
}

// NewCheckpointManager creates a new checkpoint manager
func NewCheckpointManager(maxCheckpoints int, ttl time.Duration) *CheckpointManager {
	if maxCheckpoints <= 0 {
		maxCheckpoints = 50 // Default: keep last 50 checkpoints
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour // Default: 24 hour TTL
	}

	return &CheckpointManager{
		checkpoints:    make(map[string][]*Checkpoint),
		maxCheckpoints: maxCheckpoints,
		ttl:            ttl,
	}
}

// CreateCheckpoint creates a new checkpoint for a session
func (cm *CheckpointManager) CreateCheckpoint(session *Session, description string) (*Checkpoint, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Get cookies
	var cookies []*proto.NetworkCookie
	if session.Page != nil {
		cdpCookies, err := session.Page.Cookies([]string{})
		if err == nil {
			cookies = cdpCookies
		}
	}

	// Get localStorage using simple JS
	localStorage := make(map[string]string)
	if session.Page != nil {
		if result, err := session.Page.Eval(`() => JSON.stringify(localStorage)`); err == nil {
			str := result.Value.String()
			if str != "" && str != "null" {
				_ = json.Unmarshal([]byte(str), &localStorage)
			}
		}
	}

	// Get sessionStorage
	sessionStorage := make(map[string]string)
	if session.Page != nil {
		if result, err := session.Page.Eval(`() => JSON.stringify(sessionStorage)`); err == nil {
			str := result.Value.String()
			if str != "" && str != "null" {
				_ = json.Unmarshal([]byte(str), &sessionStorage)
			}
		}
	}

	checkpoint := &Checkpoint{
		ID:             fmt.Sprintf("cp-%d", time.Now().UnixNano()),
		SessionID:      session.ID,
		Timestamp:      time.Now(),
		URL:            session.URL,
		Title:          session.Title,
		Cookies:        cookies,
		LocalStorage:   localStorage,
		SessionStorage: sessionStorage,
		ActionIndex:    0, // Will be updated by caller
		Description:    description,
	}

	// Add to session checkpoints
	sessionCheckpoints, exists := cm.checkpoints[session.ID]
	if !exists {
		sessionCheckpoints = make([]*Checkpoint, 0)
	}

	sessionCheckpoints = append(sessionCheckpoints, checkpoint)

	// Trim to max checkpoints
	if len(sessionCheckpoints) > cm.maxCheckpoints {
		sessionCheckpoints = sessionCheckpoints[len(sessionCheckpoints)-cm.maxCheckpoints:]
	}

	cm.checkpoints[session.ID] = sessionCheckpoints

	return checkpoint, nil
}

// GetLatestCheckpoint returns the most recent checkpoint for a session
func (cm *CheckpointManager) GetLatestCheckpoint(sessionID string) (*Checkpoint, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	checkpoints, exists := cm.checkpoints[sessionID]
	if !exists || len(checkpoints) == 0 {
		return nil, fmt.Errorf("no checkpoints found for session %s", sessionID)
	}

	return checkpoints[len(checkpoints)-1], nil
}

// GetCheckpointByID returns a specific checkpoint by ID
func (cm *CheckpointManager) GetCheckpointByID(sessionID, checkpointID string) (*Checkpoint, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	checkpoints, exists := cm.checkpoints[sessionID]
	if !exists {
		return nil, fmt.Errorf("no checkpoints found for session %s", sessionID)
	}

	for _, cp := range checkpoints {
		if cp.ID == checkpointID {
			return cp, nil
		}
	}

	return nil, fmt.Errorf("checkpoint %s not found", checkpointID)
}

// ListCheckpoints returns all checkpoints for a session
func (cm *CheckpointManager) ListCheckpoints(sessionID string) []*Checkpoint {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	checkpoints, exists := cm.checkpoints[sessionID]
	if !exists {
		return []*Checkpoint{}
	}

	result := make([]*Checkpoint, len(checkpoints))
	copy(result, checkpoints)
	return result
}

// RestoreFromCheckpoint restores a session from a checkpoint
func (cm *CheckpointManager) RestoreFromCheckpoint(session *Session, checkpoint *Checkpoint) error {
	session.mu.Lock()
	defer session.mu.Unlock()

	// Navigate to the checkpoint URL
	if checkpoint.URL != "" {
		if err := session.Page.Navigate(checkpoint.URL); err != nil {
			return fmt.Errorf("failed to navigate to checkpoint URL: %w", err)
		}
		session.Page.WaitLoad()
	}

	// Restore cookies
	if len(checkpoint.Cookies) > 0 {
		var cookieParams []*proto.NetworkCookieParam
		for _, c := range checkpoint.Cookies {
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
		if err := session.Page.SetCookies(cookieParams); err != nil {
			fmt.Printf("Warning: failed to restore cookies: %v\n", err)
		}
	}

	// Restore localStorage
	if len(checkpoint.LocalStorage) > 0 {
		for k, v := range checkpoint.LocalStorage {
			_, _ = session.Page.Eval(
				`(key, value) => { try { localStorage.setItem(key, value); } catch(e) { console.error(e); } }`,
				k, v)
		}
	}

	// Restore sessionStorage
	if len(checkpoint.SessionStorage) > 0 {
		for k, v := range checkpoint.SessionStorage {
			_, _ = session.Page.Eval(
				`(key, value) => { try { sessionStorage.setItem(key, value); } catch(e) { console.error(e); } }`,
				k, v)
		}
	}

	return nil
}

// DeleteCheckpoint removes a checkpoint
func (cm *CheckpointManager) DeleteCheckpoint(sessionID, checkpointID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	checkpoints, exists := cm.checkpoints[sessionID]
	if !exists {
		return fmt.Errorf("no checkpoints found for session %s", sessionID)
	}

	for i, cp := range checkpoints {
		if cp.ID == checkpointID {
			// Remove from slice
			checkpoints = append(checkpoints[:i], checkpoints[i+1:]...)
			cm.checkpoints[sessionID] = checkpoints
			return nil
		}
	}

	return fmt.Errorf("checkpoint %s not found", checkpointID)
}

// ClearCheckpoints removes all checkpoints for a session
func (cm *CheckpointManager) ClearCheckpoints(sessionID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.checkpoints, sessionID)
}

// Cleanup removes expired checkpoints
func (cm *CheckpointManager) Cleanup() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	for sessionID, checkpoints := range cm.checkpoints {
		validCheckpoints := make([]*Checkpoint, 0)
		for _, cp := range checkpoints {
			if now.Sub(cp.Timestamp) < cm.ttl {
				validCheckpoints = append(validCheckpoints, cp)
			}
		}
		if len(validCheckpoints) == 0 {
			delete(cm.checkpoints, sessionID)
		} else {
			cm.checkpoints[sessionID] = validCheckpoints
		}
	}
}

// GetCheckpointCount returns the number of checkpoints for a session
func (cm *CheckpointManager) GetCheckpointCount(sessionID string) int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.checkpoints[sessionID])
}

// SerializeCheckpoint serializes a checkpoint to JSON
func (cm *CheckpointManager) SerializeCheckpoint(cp *Checkpoint) ([]byte, error) {
	return json.Marshal(cp)
}

// DeserializeCheckpoint deserializes a checkpoint from JSON
func (cm *CheckpointManager) DeserializeCheckpoint(data []byte) (*Checkpoint, error) {
	var cp Checkpoint
	err := json.Unmarshal(data, &cp)
	return &cp, err
}
