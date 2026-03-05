package browser

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/rennaisance-jomt/axon/internal/security"
	"github.com/rennaisance-jomt/axon/pkg/logger"
)

// ActionRecord represents a recorded browser action
type ActionRecord struct {
	Type      string    `json:"type"`
	Ref       string    `json:"ref"`
	Value     string    `json:"value,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"` // success, failed, pending
	Error     string    `json:"error,omitempty"`
}

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
	RecoveryMgr   *RecoveryManager       `json:"-"` // Sprint 19: Autonomous recovery manager
	// Lifecycle monitoring
	LifecycleMonitor *LifecycleMonitor    `json:"-"` // Page lifecycle events
	TabManager     *TabManager          `json:"-"` // Multi-tab management
	Streamer       *Streamer            `json:"-"` // Sprint 27: Physical screencast streamer
	ActionHistory  []ActionRecord       `json:"action_history,omitempty"` // Sprint 27.3: Path visualization
	Vault          *security.Vault      `json:"-"` // Sprint 28: Intelligence Vault
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

// recordAction records an action in the history
func (s *Session) recordAction(actionType, ref, value, status, errStr string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record := ActionRecord{
		Type:      actionType,
		Ref:       ref,
		Value:     value,
		Timestamp: time.Now(),
		Status:    status,
		Error:     errStr,
	}

	s.ActionHistory = append(s.ActionHistory, record)

	// Keep only last 10 actions for overlay
	if len(s.ActionHistory) > 10 {
		s.ActionHistory = s.ActionHistory[len(s.ActionHistory)-10:]
	}
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
		return nil, fmt.Errorf("element not found: ref %s not in last snapshot", ref)
	}

	logger.Info("[%s] Resolving ref %s: label='%s', role='%s'", s.ID, ref, targetEl.Label, targetEl.Role)

	// Sprint 27.6: Prioritize robust JS-based search. BackendNodeID resolution (ElementFromNode) 
	// can sometimes cause CDP congestion or hangs on Windows in headless mode. 
	js := fmt.Sprintf(`
		(function() {
			var label = %s;
			var role = %s;
			var elements = document.querySelectorAll('button, [role="button"], a, input, textarea, select, [role="link"], [role="textbox"], [role="searchbox"]');
			
			// 1. Exact match pass
			for (var i = 0; i < elements.length; i++) {
				var el = elements[i];
				var text = (el.textContent || el.innerText || "").trim().toLowerCase();
				var aria = (el.getAttribute("aria-label") || "").trim().toLowerCase();
				var plac = (el.getAttribute("placeholder") || "").trim().toLowerCase();
				var val = (el.value || "").trim().toLowerCase();
				var l = label.toLowerCase();

				if (text === l || aria === l || plac === l || val === l) {
					return el;
				}
			}
			
			// 2. Partial match fallback (only for reasonably long labels)
			if (label.length > 3) {
				for (var i = 0; i < elements.length; i++) {
					var el = elements[i];
					var text = (el.textContent || "").toLowerCase();
					if (text.includes(label.toLowerCase())) return el;
				}
			}
			return null;
		})()
	`, jsonQuote(targetEl.Label), jsonQuote(targetEl.Role))

	el, err := s.Page.Timeout(5 * time.Second).ElementByJS(rod.Eval(js))
	if err == nil && el != nil {
		return el, nil
	}

	// 3. Last resort: Try BackendNodeID if JS search failed
	if targetEl.BackendNodeID > 0 {
		logger.Warn("[%s] Semantic search failed for %s, trying BackendNodeID workaround", s.ID, ref)
		node := &proto.DOMNode{BackendNodeID: targetEl.BackendNodeID}
		el, err := s.Page.Timeout(3 * time.Second).ElementFromNode(node)
		if err == nil && el != nil {
			return el, nil
		}
	}

	return nil, fmt.Errorf("could not resolve ref %s effectively", ref)
}

func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// SessionManager manages multiple browser sessions
type SessionManager struct {
	mu            sync.RWMutex
	sessions      map[string]*Session
	pool          *Pool
	maxSessionLife time.Duration
	lifecycleCh   chan string // channel for lifecycle events
	checkpointMgr *CheckpointManager // Sprint 18 & 19: For checkpoints and recovery
	vault         *security.Vault      // Sprint 28: Intelligence Vault
}

// NewSessionManager creates a new session manager
func NewSessionManager(pool *Pool, maxSessionLife time.Duration, vault *security.Vault) *SessionManager {
	if maxSessionLife <= 0 {
		maxSessionLife = 30 * time.Minute // default
	}
	return &SessionManager{
		sessions:      make(map[string]*Session),
		pool:          pool,
		maxSessionLife: maxSessionLife,
		lifecycleCh:   make(chan string, 100),
		vault:         vault,
	}
}

// NewSessionManagerWithRecovery creates a new session manager with checkpoint and recovery support
func NewSessionManagerWithRecovery(pool *Pool, maxSessionLife time.Duration, checkpointMgr *CheckpointManager, vault *security.Vault) *SessionManager {
	sm := NewSessionManager(pool, maxSessionLife, vault)
	sm.checkpointMgr = checkpointMgr
	return sm
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

	// Acquire context from pool
	browserCtx, err := sm.pool.Acquire()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire context: %w", err)
	}

	// Use the context's browser for incognito operations
	// Note: browserCtx.Context is already an incognito context
	ctx := browserCtx.Context

	// Create new page
	page, err := ctx.Page(proto.TargetCreateTarget{})
	if err != nil {
		sm.pool.Release(browserCtx)
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
		WorkerID:      browserCtx.ID,
		Context:       ctx,
		Browser:       ctx, // Context is the incognito browser
		Page:          page,
		KnownElements: make(map[string]string),
		LastElements:  make([]Element, 0),
		Vault:         sm.vault,
	}

	// Sprint 18 & 19: Initialize recovery manager if checkpoint manager exists
	if sm.checkpointMgr != nil {
		recoveryConfig := DefaultRecoveryConfig()
		session.RecoveryMgr = NewRecoveryManager(recoveryConfig, sm.checkpointMgr)
	}

	// Sprint 21: Initialize lifecycle monitor for page events
	session.LifecycleMonitor = NewLifecycleMonitor(page)

	// Sprint 21: Initialize tab manager for multi-tab support
	session.TabManager = NewTabManager(ctx)

	// Sprint 27: Initialize visual streamer
	session.Streamer = NewStreamer(page, id)
	session.Streamer.SetMetadataFunc(session.GetElementCoordinates)

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

	logger.System("SESSION_DELETE: Starting deletion of session %s", id)

	// Close the page first
	if session.Page != nil {
		logger.System("SESSION_DELETE: Closing page for session %s", id)
		session.Page.Close()
	}

	// Close the browser context (incognito)
	if session.Context != nil {
		logger.System("SESSION_DELETE: Closing browser context for session %s", id)
		session.Context.Close()
	}

	// Immediately destroy the context instead of releasing it back to pool
	// This ensures Chromium is properly terminated immediately
	if session.WorkerID != "" {
		logger.System("SESSION_DELETE: Immediately destroying context %s", session.WorkerID)
		browserCtx, err := sm.pool.GetContext(session.WorkerID)
		if err == nil {
			// Mark as closed and immediately destroy
			browserCtx.mu.Lock()
			browserCtx.Status = ContextStatusClosed
			browserCtx.mu.Unlock()
			
			// Call destroy directly to ensure immediate cleanup
			sm.pool.destroyContext(browserCtx)
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
	// If wait_until is "none", use a non-blocking JS jump to avoid rod hangs
	if waitUntil == "none" {
		_, err := s.Page.Eval(fmt.Sprintf("window.location.href = '%s'", url))
		if err != nil {
			// Fallback to standard if JS fails
			if err := s.Page.Navigate(url); err != nil {
				return fmt.Errorf("navigation fallback failed: %w", err)
			}
		}
	} else {
		if err := s.Page.Navigate(url); err != nil {
			return fmt.Errorf("navigation failed: %w", err)
		}
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
	case "none":
		// Return immediately
	default:
		// Default to load with timeout
		errCh := make(chan error, 1)
		go func() {
			errCh <- s.Page.WaitLoad()
		}()
		
		select {
		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("wait load failed: %w", err)
			}
		case <-time.After(15 * time.Second):
			logger.Warn("[%s] WaitLoad timed out, continuing anyway", s.ID)
		}
	}

	// Update metadata
	s.mu.Lock()
	s.URL = url
	s.Status = "active"
	s.mu.Unlock()

	// Get title with retry
	for i := 0; i < 5; i++ {
		if res, err := s.Page.Eval("document.title"); err == nil {
			title := res.Value.String()
			if title != "" {
				s.mu.Lock()
				s.Title = title
				s.mu.Unlock()
				break
			}
		}
	}
	// Record navigation
	s.recordAction("navigate", url, "", "success", "")
	logger.Success("[%s] Navigation to %s completed", s.ID, url)

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

	logger.Action("[%s] Clicking element: %s", s.ID, selector)
	
	// Sprint 27.5: Perform browser interaction OUTSIDE of the session lock to prevent deadlocks
	// and server-wide hangs during slow page responses.
	// We use the direct rod.Element.Timeout() pattern.
	err = element.Timeout(15 * time.Second).Click(proto.InputMouseButtonLeft, 1)
	
	status := "success"
	errStr := ""
	if err != nil {
		status = "failed"
		errStr = err.Error()
	}

	s.mu.Lock()
	s.LastAction = time.Now()
	s.mu.Unlock()

	s.recordAction("click", selector, "", status, errStr)
	return err
}


// Fill fills an input field
func (s *Session) Fill(selector string, value string) error {
	element, err := s.resolveSelector(selector)
	if err != nil {
		return err
	}

	// Sprint 28: Intelligence Vault Injection
	if strings.HasPrefix(value, "@vault:") {
		parts := strings.Split(strings.TrimPrefix(value, "@vault:"), ":")
		secretName := parts[0]
		fieldName := "password" // default
		if len(parts) > 1 {
			fieldName = parts[1]
		}

		logger.Action("[%s] Injecting secret '%s' (field: %s) from vault into %s", s.ID, secretName, fieldName, selector)

		if s.Vault == nil {
			return fmt.Errorf("vault not initialized for this session")
		}

		secret, err := s.Vault.GetSecret(secretName, s.URL)
		if err != nil {
			logger.Error("[%s] Vault injection failed: %v", s.ID, err)
			return fmt.Errorf("vault access denied: %w", err)
		}

		// Inject the correct field
		switch strings.ToLower(fieldName) {
		case "username":
			value = secret.Username
		case "password":
			value = secret.Password
		case "value":
			value = secret.Value
		default:
			if val, ok := secret.Metadata[fieldName]; ok {
				value = val
			} else {
				return fmt.Errorf("field '%s' not found in secret '%s'", fieldName, secretName)
			}
		}

		// Sprint 28: Security Guards
		// The correct rod way to pass an element to page-level JS is via JSArgs.
		// When a JSArgs entry is *proto.RuntimeRemoteObject, rod passes it through
		// CDP's callFunctionOn JSArgs, and it arrives as the first parameter in JS.
		// This works regardless of how the element handle was obtained (CSS, JS eval,
		// or BackendNodeID), because we're operating at the CDP object level.
		if element.Object != nil {
			// Anti-Phishing Guard
			phishingRes, phishErr := s.Page.Evaluate(rod.Eval(`(el) => {
				if (!el) return "";
				var form = el.closest('form');
				return form ? form.action : "";
			}`, element.Object))

			if phishErr == nil && phishingRes != nil {
				actionURL := phishingRes.Value.Str()
				if actionURL != "" {
					if parsedAction, parseErr := url.Parse(actionURL); parseErr == nil && parsedAction.Host != "" {
						actionBase := security.GetBaseDomain(actionURL)
						currentBase := security.GetBaseDomain(s.URL)
						if actionBase != currentBase {
							logger.Error("[%s] Phishing Guard Blocked Injection: form action '%s' is cross-origin vs page '%s'", s.ID, actionURL, s.URL)
							s.recordAction("fill", selector, "[BLOCKED_BY_GUARD]", "failed", "untrusted cross-origin target")
							return fmt.Errorf("phishing protection triggered: form submits to untrusted cross-origin domain %s", actionBase)
						}
					}
				}
			}

			// DOM Masking: protect against session replay leaks by changing
			// the input type to 'password' and marking it with data-axon-masked.
			_, _ = s.Page.Evaluate(rod.Eval(`(el) => {
				if (el && el.tagName === 'INPUT' && el.type !== 'password') {
					el.setAttribute('data-axon-masked', 'true');
					el.type = 'password';
				}
			}`, element.Object))
		} else {
			logger.Warn("[%s] Vault injection: element.Object is nil, skipping security guards", s.ID)
		}

		// Note: We don't record the actual secret value in action history
		s.recordAction("fill", selector, "[SECRET_INJECTED]", "success", "")
	} else {
		logger.Action("[%s] Filling element %s with value", s.ID, selector)
		s.recordAction("fill", selector, value, "success", "")
	}

	// Sprint 27.6: Use Input for standard rod.Element behavior.
	err = element.Timeout(5 * time.Second).Input(value)

	if err != nil {
		logger.Error("[%s] Fill failed: %v", s.ID, err)
		// Update status if it was success before
		for i := len(s.ActionHistory) - 1; i >= 0; i-- {
			if s.ActionHistory[i].Ref == selector && s.ActionHistory[i].Type == "fill" {
				s.ActionHistory[i].Status = "failed"
				s.ActionHistory[i].Error = err.Error()
				break
			}
		}
	}

	s.mu.Lock()
	s.LastAction = time.Now()
	s.mu.Unlock()

	return err
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

// GetElementCoordinates returns the viewport coordinates for all elements in the last snapshot
func (s *Session) GetElementCoordinates() map[string]interface{} {
	s.mu.RLock()
	elements := s.LastElements
	s.mu.RUnlock()

	result := make(map[string]interface{})
	var coords []map[string]interface{}

	for _, el := range elements {
		if el.BackendNodeID == 0 {
			continue
		}

		// Get box model
		box, err := proto.DOMGetBoxModel{
			BackendNodeID: el.BackendNodeID,
		}.Call(s.Page)

		if err == nil && box != nil {
			// Use the border box as the element's hit area
			// box.Border is [x1, y1, x2, y2, x3, y3, x4, y4]
			if len(box.Model.Border) >= 8 {
				x := box.Model.Border[0]
				y := box.Model.Border[1]
				width := box.Model.Border[2] - box.Model.Border[0]
				height := box.Model.Border[5] - box.Model.Border[1]

				coords = append(coords, map[string]interface{}{
					"ref":    el.Ref,
					"label":  el.Label,
					"type":   el.Type,
					"x":      x,
					"y":      y,
					"width":  width,
					"height": height,
				})
			}
		}
	}

	result["elements"] = coords
	result["url"] = s.URL
	result["title"] = s.Title
	s.mu.RLock()
	result["action_history"] = s.ActionHistory
	s.mu.RUnlock()
	return result
}

// Press presses a key
func (s *Session) Press(selector string, key string) error {
	element, err := s.resolveSelector(selector)
	if err != nil {
		return err
	}

	logger.Action("[%s] Pressing key %s on element %s", s.ID, key, selector)
	
	switch strings.ToLower(key) {
	case "enter":
		err = element.Timeout(10 * time.Second).Type(input.Enter)
	case "escape", "esc":
		err = element.Timeout(10 * time.Second).Type(input.Escape)
	case "tab":
		err = element.Timeout(10 * time.Second).Type(input.Tab)
	case "backspace":
		err = element.Timeout(10 * time.Second).Type(input.Backspace)
	default:
		// Fallback to literal input if not a control key
		err = element.Timeout(10 * time.Second).Input(key)
	}

	status := "success"
	errStr := ""
	if err != nil {
		status = "failed"
		errStr = err.Error()
	}

	s.mu.Lock()
	s.LastAction = time.Now()
	s.mu.Unlock()

	s.recordAction("press", selector, key, status, errStr)
	return err
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

			// Check if context is still healthy
			if s.WorkerID != "" {
				ctx, err := sm.pool.GetContext(s.WorkerID)
				if err != nil || ctx.Status != ContextStatusHealthy {
					logger.Error("Session %s context %s is unhealthy, recreating session", session.ID, s.WorkerID)
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

	// Acquire new context from pool (already incognito)
	browserCtx, err := sm.pool.Acquire()
	if err != nil {
		logger.Error("Failed to acquire new context for session %s: %v", oldSession.ID, err)
		sm.CloseSession(oldSession.ID)
		return
	}

	// Use the context directly (already incognito)
	ctx := browserCtx.Context

	page, err := ctx.Page(proto.TargetCreateTarget{})
	if err != nil {
		logger.Error("Failed to create page for session %s: %v", oldSession.ID, err)
		sm.pool.Release(browserCtx)
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

		// Release old context
		if oldWorkerID != "" {
			if oldCtx, err := sm.pool.GetContext(oldWorkerID); err == nil {
				sm.pool.Release(oldCtx)
			}
		}

		// Update with new resources
		session.WorkerID = browserCtx.ID
		session.Context = ctx
		session.Browser = ctx
		session.Page = page
		logger.Success("Session %s migrated to context %s", oldSession.ID, browserCtx.ID)
	}
	sm.mu.Unlock()
}
