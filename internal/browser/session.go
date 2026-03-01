package browser

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/rennaisance-jomt/axon/pkg/logger"
)

// generateSessionID generates a unique session ID
func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// Session represents a browser session
type Session struct {
	ID            string                 `json:"session_id"`
	Status        string                 `json:"status"` // created, active, idle, closed
	Profile       string                 `json:"profile,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	LastAction    time.Time              `json:"last_action,omitempty"`
	URL           string                 `json:"url,omitempty"`
	Title         string                 `json:"title,omitempty"`
	AuthState     string                 `json:"auth_state,omitempty"` // unknown, logged_in, logged_out
	PageState     string                 `json:"page_state,omitempty"` // loading, ready, error
	WorkerID      string                 `json:"worker_id,omitempty"`
	Context       *rod.Browser          `json:"-"`
	Browser       *rod.Browser          `json:"-"`
	Page          *rod.Page             `json:"-"`
	KnownElements map[string]string     `json:"known_elements,omitempty"` // Map intent to selector
	LastElements  []Element              `json:"-"` // Last snapshot elements for action lookup
	mu            sync.RWMutex
}

// resolveSelector resolves a selector or ref to a rod.Element
func (s *Session) resolveSelector(selector string) (*rod.Element, error) {
	// If it's a ref selector like [data-ref='b38'], resolve via ref
	if strings.HasPrefix(selector, "[data-ref='") && strings.HasSuffix(selector, "']") {
		ref := strings.TrimPrefix(selector, "[data-ref='")
		ref = strings.TrimSuffix(ref, "']")
		return s.getElementFromRef(ref)
	}

	// Otherwise use standard CSS selector with timeout
	return s.Page.Timeout(5 * time.Second).Element(selector)
}

// getElementFromRef finds an element by its semantic reference
func (s *Session) getElementFromRef(ref string) (*rod.Element, error) {
	s.mu.RLock()
	var targetEl Element
	for _, el := range s.LastElements {
		if el.Ref == ref {
			targetEl = el
			break
		}
	}
	s.mu.RUnlock()

	if targetEl.Ref == "" {
		return nil, fmt.Errorf("element not found: ref %s not in last snapshot. Run snapshot first.", ref)
	}

	// 1. Try resolving via BackendNodeID (Most Robust)
	if targetEl.BackendNodeID > 0 {
		
		// Map BackendNodeID to rod.Element
		node := &proto.DOMNode{
			BackendNodeID: targetEl.BackendNodeID,
		}
		
		el, err := s.Page.ElementFromNode(node)
		if err == nil && el != nil {
			return el, nil
		}
		logger.Warn("[%s] BackendNodeID resolution failed for ref %s: %v. Falling back to semantic search.", s.ID, ref, err)
	}

	// 2. Fallback to Semantic Search (Text/Label)
	logger.Info("[%s] Performing semantic search for ref %s: label=%s", s.ID, ref, targetEl.Label)

	// JS-based search for various interactive elements matching the label
	js := fmt.Sprintf(`
		(function() {
			var label = '%s';
			var elements = document.querySelectorAll('button, [role="button"], a, input, textarea, select, [role="link"], [role="textbox"]');
			for (var i = 0; i < elements.length; i++) {
				var el = elements[i];
				var text = (el.textContent || el.innerText || "").trim();
				var ariaLabel = (el.getAttribute("aria-label") || "").trim();
				var title = (el.getAttribute("title") || "").trim();
				var placeholder = (el.getAttribute("placeholder") || "").trim();
				var value = (el.value || "").trim();

				if (text.toLowerCase() === label.toLowerCase() ||
				    ariaLabel.toLowerCase() === label.toLowerCase() ||
				    title.toLowerCase() === label.toLowerCase() ||
				    placeholder.toLowerCase() === label.toLowerCase() ||
				    (el.type === "submit" && value.toLowerCase() === label.toLowerCase())) {
					return el;
				}
				
				// Partial match as secondary fallback
				if (text.toLowerCase().includes(label.toLowerCase()) && label.length > 3) {
					return el;
				}
			}
			return null;
		})();
	`, targetEl.Label)

	// Using rod's built-in support for resolving elements from JS
	return s.Page.ElementByJS(rod.Eval(js))
}

// SessionManager manages multiple browser sessions
type SessionManager struct {
	mu            sync.RWMutex
	sessions      map[string]*Session
	pool          *Pool
	maxSessionLife time.Duration
	lifecycleCh   chan string // channel for lifecycle events
}

// NewSessionManager creates a new session manager
func NewSessionManager(pool *Pool, maxSessionLife time.Duration) *SessionManager {
	if maxSessionLife <= 0 {
		maxSessionLife = 30 * time.Minute // default
	}
	return &SessionManager{
		sessions:      make(map[string]*Session),
		pool:          pool,
		maxSessionLife: maxSessionLife,
		lifecycleCh:   make(chan string, 100),
	}
}

// Create creates a new session
func (sm *SessionManager) Create(id string, profile string) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Generate unique ID if not provided
	if id == "" {
		id = generateSessionID()
	}

	if _, exists := sm.sessions[id]; exists {
		return nil, fmt.Errorf("session %s already exists", id)
	}

	// Acquire worker from pool
	worker, err := sm.pool.Acquire()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire worker: %w", err)
	}

	// Create browser context using the worker's browser
	worker.mu.RLock()
	browser := worker.Browser
	worker.mu.RUnlock()

	// Create incognito browser (acts as a context)
	incognito, err := browser.Incognito()
	if err != nil {
		sm.pool.Release(worker)
		return nil, fmt.Errorf("failed to create incognito: %w", err)
	}

	// Create new page
	page, err := incognito.Page(proto.TargetCreateTarget{})
	if err != nil {
		sm.pool.Release(worker)
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
		WorkerID:      worker.ID,
		Context:       incognito,
		Browser:       browser,
		Page:          page,
		KnownElements: make(map[string]string),
		LastElements:  make([]Element, 0),
	}

	// If profile is provided, import cookies
	if profile != "" {
		if err := session.ImportCookies(profile); err != nil {
			// Log but don't fail - cookies might be invalid
			logger.Warn("[%s] Failed to import cookies from profile %s: %v", id, profile, err)
		}
	}

	sm.sessions[id] = session

	// Start lifecycle monitor for this session
	go sm.monitorSession(session)

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

	// Close the page first
	if session.Page != nil {
		session.Page.Close()
	}

	// Close the browser context (incognito)
	if session.Context != nil {
		session.Context.Close()
	}

	// Release worker back to pool
	if session.WorkerID != "" {
		worker, err := sm.pool.GetWorker(session.WorkerID)
		if err == nil {
			sm.pool.Release(worker)
		}
	}

	delete(sm.sessions, id)
	logger.Success("Session %s deleted and browser resources released", id)
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

// Click clicks on an element
func (s *Session) Click(selector string) error {
	element, err := s.resolveSelector(selector)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	logger.Action("[%s] Clicking element: %s", s.ID, selector)
	element.MustWaitVisible()
	element.MustWaitStable()
	return element.Click(proto.InputMouseButtonLeft, 1)
}


// Fill fills an input field
func (s *Session) Fill(selector string, value string) error {
	element, err := s.resolveSelector(selector)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	logger.Action("[%s] Filling element %s with value", s.ID, selector)
	element.MustWaitVisible()
	element.MustWaitStable()
	return element.Input(value)
}

// SetLastElements stores the elements from the last snapshot
func (s *Session) SetLastElements(elements []Element) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastElements = elements
}

// GetLastElements retrieves the elements from the last snapshot
func (s *Session) GetLastElements() []Element {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastElements
}

// Press presses a key
func (s *Session) Press(selector string, key string) error {
	element, err := s.resolveSelector(selector)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

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

// monitorSession monitors a session for lifecycle events
func (sm *SessionManager) monitorSession(session *Session) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.mu.RLock()
			s, exists := sm.sessions[session.ID]
			sm.mu.RUnlock()

			if !exists || s.Status == "closed" {
				return
			}

			// Check if session has exceeded max lifetime
			if time.Since(s.CreatedAt) > sm.maxSessionLife {
				logger.Warn("Session %s exceeded max lifetime, closing autonomously", session.ID)
				sm.CloseSession(session.ID)
				return
			}

			// Check if worker is still healthy
			if s.WorkerID != "" {
				worker, err := sm.pool.GetWorker(s.WorkerID)
				if err != nil || worker.Status != WorkerStatusHealthy {
					logger.Error("Session %s worker %s is unhealthy, recreating session", session.ID, s.WorkerID)
					// Trigger session recreation
					sm.recreateSession(session)
					return
				}
			}
		}
	}
}

// CloseSession gracefully closes a session
func (sm *SessionManager) CloseSession(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[id]
	if !exists {
		return fmt.Errorf("session %s not found", id)
	}

	session.mu.Lock()
	session.Status = "closed"
	session.mu.Unlock()

	if session.Page != nil {
		session.Page.Close()
	}
	if session.Context != nil {
		session.Context.Close()
	}

	// Release worker
	if session.WorkerID != "" {
		worker, err := sm.pool.GetWorker(session.WorkerID)
		if err == nil {
			sm.pool.Release(worker)
		}
	}

	delete(sm.sessions, id)
	logger.Success("Session %s closed (max lifetime exceeded)", id)
	return nil
}

// recreateSession recreates a session on a new worker
func (sm *SessionManager) recreateSession(oldSession *Session) {
	oldWorkerID := oldSession.WorkerID

	// Get worker
	worker, err := sm.pool.Acquire()
	if err != nil {
		logger.Error("Failed to acquire new worker for session %s: %v", oldSession.ID, err)
		sm.CloseSession(oldSession.ID)
		return
	}

	// Create new browser context
	worker.mu.RLock()
	browser := worker.Browser
	worker.mu.RUnlock()

	incognito, err := browser.Incognito()
	if err != nil {
		logger.Error("Failed to create incognito for session %s: %v", oldSession.ID, err)
		sm.pool.Release(worker)
		sm.CloseSession(oldSession.ID)
		return
	}

	page, err := incognito.Page(proto.TargetCreateTarget{})
	if err != nil {
		logger.Error("Failed to create page for session %s: %v", oldSession.ID, err)
		sm.pool.Release(worker)
		sm.CloseSession(oldSession.ID)
		return
	}

	// Set up network blocking
	router := page.HijackRequests()
	router.MustAdd("*", func(ctx *rod.Hijack) {
		reqType := ctx.Request.Type()
		urlStr := ctx.Request.URL().String()

		if reqType == proto.NetworkResourceTypeImage ||
			reqType == proto.NetworkResourceTypeMedia ||
			reqType == proto.NetworkResourceTypeFont ||
			reqType == proto.NetworkResourceTypeStylesheet {
			ctx.Response.Fail(proto.NetworkErrorReasonBlockedByClient)
			return
		}

		if strings.Contains(urlStr, "google-analytics.com") ||
			strings.Contains(urlStr, "doubleclick.net") ||
			strings.Contains(urlStr, "facebook.net") ||
			strings.Contains(urlStr, "clarity.ms") ||
			strings.HasSuffix(urlStr, ".woff2") {
			ctx.Response.Fail(proto.NetworkErrorReasonBlockedByClient)
			return
		}

		ctx.ContinueRequest(&proto.FetchContinueRequest{})
	})
	go router.Run()

	// Update session
	sm.mu.Lock()
	if session, exists := sm.sessions[oldSession.ID]; exists {
		// Close old resources
		if oldSession.Page != nil {
			oldSession.Page.Close()
		}
		if oldSession.Context != nil {
			oldSession.Context.Close()
		}

		// Release old worker
		if oldWorkerID != "" {
			if oldWorker, err := sm.pool.GetWorker(oldWorkerID); err == nil {
				sm.pool.Release(oldWorker)
			}
		}

		// Update with new resources
		session.WorkerID = worker.ID
		session.Context = incognito
		session.Browser = browser
		session.Page = page
		logger.Success("Session %s migrated to worker %s", oldSession.ID, worker.ID)
	}
	sm.mu.Unlock()
}
