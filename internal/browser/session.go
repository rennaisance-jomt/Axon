package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// Session represents a browser session
type Session struct {
	ID          string                 `json:"session_id"`
	Status     string                 `json:"status"` // created, active, idle, closed
	Profile    string                 `json:"profile,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	LastAction time.Time              `json:"last_action,omitempty"`
	URL        string                 `json:"url,omitempty"`
	Title      string                 `json:"title,omitempty"`
	AuthState  string                 `json:"auth_state,omitempty"` // unknown, logged_in, logged_out
	PageState  string                 `json:"page_state,omitempty"` // loading, ready, error
	Context    *rod.BrowserContext   `json:"-"`
	Browser    *rod.Browser          `json:"-"`
	Page       *rod.Page             `json:"-"`
	mu         sync.RWMutex
}

// SessionManager manages multiple browser sessions
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	pool     *Pool
}

// NewSessionManager creates a new session manager
func NewSessionManager(pool *Pool) *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
		pool:     pool,
	}
}

// Create creates a new session
func (sm *SessionManager) Create(id string, profile string) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.sessions[id]; exists {
		return nil, fmt.Errorf("session %s already exists", id)
	}

	// Acquire browser from pool
	browser, err := sm.pool.Acquire()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire browser: %w", err)
	}

	// Create browser context
	var ctxOpts []rod.BrowserContextOption
	if profile != "" {
		ctxOpts = append(ctxOpts, rod.BrowserContextOptionStorageState(profile))
	}

	ctx, err := browser.CreateContext(ctxOpts...)
	if err != nil {
		sm.pool.Release(browser)
		return nil, fmt.Errorf("failed to create context: %w", err)
	}

	// Create new page
	page, err := ctx.CreatePage()
	if err != nil {
		ctx.Close()
		sm.pool.Release(browser)
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	session := &Session{
		ID:         id,
		Status:     "created",
		Profile:    profile,
		CreatedAt:  time.Now(),
		LastAction: time.Now(),
		Context:    ctx,
		Browser:    browser,
		Page:       page,
	}

	sm.sessions[id] = session
	return session, nil
}

// Get retrieves a session by ID
func (sm *SessionManager) Get(id string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[id]
	if !exists {
		return nil, fmt.Errorf("session %s not found", id)
	}

	if session.Status == "closed" {
		return nil, fmt.Errorf("session %s is closed", id)
	}

	return session, nil
}

// List returns all sessions
func (sm *SessionManager) List() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

// Delete closes and removes a session
func (sm *SessionManager) Delete(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[id]
	if !exists {
		return fmt.Errorf("session %s not found", id)
	}

	if session.Page != nil {
		session.Page.Close()
	}
	if session.Context != nil {
		session.Context.Close()
	}

	// Release browser back to pool
	sm.pool.Release(session.Browser)

	delete(sm.sessions, id)
	return nil
}

// Update updates session metadata
func (sm *SessionManager) Update(session *Session) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	session.LastAction = time.Now()

	if s, exists := sm.sessions[session.ID]; exists {
		s.Status = session.Status
		s.URL = session.URL
		s.Title = session.Title
		s.AuthState = session.AuthState
		s.PageState = session.PageState
		s.LastAction = session.LastAction
	}
}

// Navigate navigates to a URL
func (s *Session) Navigate(url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.Page.Navigate(url); err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}

	// Wait for load
	if err := s.Page.WaitLoad(); err != nil {
		return fmt.Errorf("wait load failed: %w", err)
	}

	// Update metadata
	s.URL = url
	s.Status = "active"

	// Get title
	if title, err := s.Page.Title(); err == nil {
		s.Title = title
	}

	return nil
}

// NavigateBack goes back in history
func (s *Session) NavigateBack() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.Page.GoBack(); err != nil {
		return fmt.Errorf("go back failed: %w", err)
	}
	s.Page.WaitLoad()
	return nil
}

// NavigateForward goes forward in history
func (s *Session) NavigateForward() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.Page.GoForward(); err != nil {
		return fmt.Errorf("go forward failed: %w", err)
	}
	s.Page.WaitLoad()
	return nil
}

// Reload reloads the page
func (s *Session) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.Page.Reload(); err != nil {
		return fmt.Errorf("reload failed: %w", err)
	}
	s.Page.WaitLoad()
	return nil
}

// Click clicks an element
func (s *Session) Click(selector string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	element, err := s.Page.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found: %w", err)
	}

	return element.Click()
}

// Fill fills an input field
func (s *Session) Fill(selector string, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	element, err := s.Page.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found: %w", err)
	}

	return element.Input(value)
}

// Press presses a key
func (s *Session) Press(selector string, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	element, err := s.Page.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found: %w", err)
	}

	return element.Type(key)
}

// Screenshot takes a screenshot
func (s *Session) Screenshot(fullPage bool) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.Page.Screenshot(fullPage, &proto.PageCaptureScreenshot{})
}

// GetCookies gets all cookies
func (s *Session) GetCookies() ([]*proto.NetworkCookie, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.Page.Cookies()
}

// SetCookies sets cookies
func (s *Session) SetCookies(cookies []*proto.NetworkCookieParam) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.Page.SetCookies(cookies)
}

// GetSessionJSON returns session as JSON
func (sm *SessionManager) GetSessionJSON(id string) ([]byte, error) {
	session, err := sm.Get(id)
	if err != nil {
		return nil, err
	}

	return json.Marshal(session)
}
