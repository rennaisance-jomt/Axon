package browser

import (
	"encoding/json"
	"fmt"
	"strings"
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
	Context    *rod.Browser          `json:"-"`
	Browser    *rod.Browser          `json:"-"`
	Page           *rod.Page             `json:"-"`
	KnownElements  map[string]string     `json:"known_elements,omitempty"` // Map intent to selector
	mu             sync.RWMutex
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
	// Create incognito browser (acts as a context)
	incognito, err := browser.Incognito()
	if err != nil {
		sm.pool.Release(browser)
		return nil, fmt.Errorf("failed to create incognito: %w", err)
	}

	// Create new page
	page, err := incognito.Page(proto.TargetCreateTarget{})
	if err != nil {
		sm.pool.Release(browser)
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	// Sprint 4: Headless-Native Network Blocking
	router := page.HijackRequests()
	router.MustAdd("*", func(ctx *rod.Hijack) {
		reqType := ctx.Request.Type()
		urlStr := ctx.Request.URL().String()

		// Aggressively drop visual assets to slash page load time
		if reqType == proto.NetworkResourceTypeImage ||
			reqType == proto.NetworkResourceTypeMedia ||
			reqType == proto.NetworkResourceTypeFont ||
			reqType == proto.NetworkResourceTypeStylesheet {
			ctx.Response.Fail(proto.NetworkErrorReasonBlockedByClient)
			return
		}

		// Analytics and known heavy trackers blocklist
		if strings.Contains(urlStr, "google-analytics.com") ||
			strings.Contains(urlStr, "doubleclick.net") ||
			strings.Contains(urlStr, "facebook.net") ||
			strings.Contains(urlStr, "clarity.ms") ||
			strings.HasSuffix(urlStr, ".woff2") {
			ctx.Response.Fail(proto.NetworkErrorReasonBlockedByClient)
			return
		}

		// Allow all other requests (Fetch, XHR, Document, JS)
		ctx.ContinueRequest(&proto.FetchContinueRequest{})
	})
	go router.Run()

	session := &Session{
		ID:            id,
		Status:        "created",
		Profile:       profile,
		CreatedAt:     time.Now(),
		LastAction:    time.Now(),
		Context:       incognito,
		Browser:       browser,
		Page:          page,
		KnownElements: make(map[string]string),
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

// Navigate navigates to a URL with optional wait condition
func (s *Session) Navigate(url string, waitUntil string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.Page.Navigate(url); err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}

	// Wait based on condition
	switch waitUntil {
	case "networkidle":
		_ = proto.DOMEnable{}.Call(s.Page)
		_ = proto.AnimationEnable{}.Call(s.Page)
		eventFired := make(chan bool, 1)
		
		go func() {
			waitDOM := s.Page.WaitEvent(&proto.DOMChildNodeInserted{})
			waitDOM()
			eventFired <- true
		}()

		go func() {
			waitAnim := s.Page.WaitEvent(&proto.AnimationAnimationCanceled{})
			waitAnim()
			eventFired <- true
		}()

		go func() {
			_ = s.Page.WaitIdle(1 * time.Second)
		}()
		
		select {
		case <-eventFired:
		case <-time.After(10 * time.Second):
		}
	case "domcontentloaded":
		wait := s.Page.WaitEvent(&proto.PageDomContentEventFired{})
		wait()
	default:
		// Default to load
		if err := s.Page.WaitLoad(); err != nil {
			return fmt.Errorf("wait load failed: %w", err)
		}
	}

	// Update metadata
	s.URL = url
	s.Status = "active"

	// Get title with retry
	for i := 0; i < 5; i++ {
		if res, err := s.Page.Eval("document.title"); err == nil {
			title := res.Value.String()
			if title != "" {
				s.Title = title
				break
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	return nil
}

// Resize resizes the browser window
func (s *Session) Resize(width, height int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.Page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  width,
		Height: height,
	})
}

// NavigateBack goes back in history
func (s *Session) NavigateBack() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.Page.Eval("window.history.back()"); err != nil {
		return fmt.Errorf("go back failed: %w", err)
	}
	_ = s.Page.WaitLoad()
	return nil
}

// NavigateForward goes forward in history
func (s *Session) NavigateForward() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.Page.Eval("window.history.forward()"); err != nil {
		return fmt.Errorf("go forward failed: %w", err)
	}
	_ = s.Page.WaitLoad()
	return nil
}

// Reload reloads the page
func (s *Session) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.Page.Reload(); err != nil {
		return fmt.Errorf("reload failed: %w", err)
	}
	_ = s.Page.WaitLoad()
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
	element.MustWaitVisible().MustWaitStable()

	return element.Click(proto.InputMouseButtonLeft, 1)
}

// Fill fills an input field
func (s *Session) Fill(selector string, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	element, err := s.Page.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found: %w", err)
	}
	element.MustWaitVisible().MustWaitStable()

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
	element.MustWaitVisible().MustWaitStable()

	return element.Input(key)
}

// Screenshot takes a screenshot
func (s *Session) Screenshot(fullPage bool) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if fullPage {
		return s.FullPageScreenshot()
	}

	return s.Page.Screenshot(false, &proto.PageCaptureScreenshot{})
}

// FullPageScreenshot takes a robust full-page screenshot using Chromium's native capture metrics
func (s *Session) FullPageScreenshot() ([]byte, error) {

	// Use rod's native full-page screenshot which leverages 
	// Page.getLayoutMetrics and Emulation.setDeviceMetricsOverride
	// to ensure the entire content is captured in one pass.
	// This is more standard and robust than manual scrolling.
	return s.Page.Screenshot(true, &proto.PageCaptureScreenshot{
		Format:                proto.PageCaptureScreenshotFormatPng,
		FromSurface:           true,
		CaptureBeyondViewport: true,
	})
}

// GetSessionJSON returns session as JSON
func (sm *SessionManager) GetSessionJSON(id string) ([]byte, error) {
	session, err := sm.Get(id)
	if err != nil {
		return nil, err
	}

	return json.Marshal(session)
}
