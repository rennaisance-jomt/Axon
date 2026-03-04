package browser

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// LifecycleEvent represents a page lifecycle event
type LifecycleEvent struct {
	Name        string `json:"name"`         // Event name
	Timestamp   int64  `json:"timestamp_ms"` // Milliseconds since epoch
}

// NavigationState holds navigation and lifecycle information
type NavigationState struct {
	URL                string           `json:"url"`
	Title              string           `json:"title"`
	LoadState          string           `json:"load_state"` // pending, loading, complete
	LifecycleEvents    []LifecycleEvent `json:"lifecycle_events"`
	FirstContentfulPaint int64         `json:"fcp_ms,omitempty"`
	FirstMeaningfulPaint int64         `json:"fmp_ms,omitempty"`
	DomContentLoaded   int64           `json:"dom_content_loaded_ms"`
	LoadComplete       int64           `json:"load_complete_ms"`
	NetworkIdle        int64           `json:"network_idle_ms,omitempty"`
	Error              string           `json:"error,omitempty"`
}

// LifecycleMonitor monitors page lifecycle events
type LifecycleMonitor struct {
	mu           sync.RWMutex
	enabled      bool
	page         *rod.Page
	events       []LifecycleEvent
	startTime    time.Time
	listeners    map[string]chan LifecycleEvent
}

// NewLifecycleMonitor creates a new lifecycle monitor
func NewLifecycleMonitor(page *rod.Page) *LifecycleMonitor {
	return &LifecycleMonitor{
		page:      page,
		events:    make([]LifecycleEvent, 0),
		listeners: make(map[string]chan LifecycleEvent),
	}
}

// Enable turns on lifecycle event monitoring
func (lm *LifecycleMonitor) Enable() error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if lm.enabled {
		return nil
	}

	// Enable lifecycle events in CDP
	err := proto.PageSetLifecycleEventsEnabled{
		Enabled: true,
	}.Call(lm.page)

	if err != nil {
		return fmt.Errorf("failed to enable lifecycle events: %w", err)
	}

	lm.enabled = true
	lm.startTime = time.Now()

	// Subscribe to lifecycle events
	go lm.subscribeToLifecycleEvents()

	return nil
}

// Disable turns off lifecycle event monitoring
func (lm *LifecycleMonitor) Disable() error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if !lm.enabled {
		return nil
	}

	err := proto.PageSetLifecycleEventsEnabled{
		Enabled: false,
	}.Call(lm.page)

	lm.enabled = false
	return err
}

// subscribeToLifecycleEvents sets up event listeners
func (lm *LifecycleMonitor) subscribeToLifecycleEvents() {
	// This is handled by rod's event system
	// We'll use polling as a simpler approach
}

// GetEvents returns all captured lifecycle events
func (lm *LifecycleMonitor) GetEvents() []LifecycleEvent {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	events := make([]LifecycleEvent, len(lm.events))
	copy(events, lm.events)
	return events
}

// GetState returns current navigation state
func (lm *LifecycleMonitor) GetState() *NavigationState {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	state := &NavigationState{
		LifecycleEvents: make([]LifecycleEvent, len(lm.events)),
	}
	copy(state.LifecycleEvents, lm.events)

	// Get page info
	if lm.page != nil {
		if info, err := lm.page.Info(); err == nil {
			state.URL = info.URL
		}
		// Get title via CDP
		titleResult, _ := proto.RuntimeEvaluate{Expression: "document.title", ReturnByValue: true}.Call(lm.page)
		if titleResult != nil && titleResult.Result.Type == "string" {
			state.Title = titleResult.Result.Value.String()
		}
	}

	// Calculate timing from events
	for _, e := range lm.events {
		switch e.Name {
		case "domContentLoaded":
			state.DomContentLoaded = e.Timestamp
			state.LoadState = "loading"
		case "commit":
			state.LoadState = "loading"
		case "load":
			state.LoadComplete = e.Timestamp
			state.LoadState = "complete"
		case "networkIdle":
			state.NetworkIdle = e.Timestamp
		case "firstContentfulPaint":
			state.FirstContentfulPaint = e.Timestamp
		case "firstMeaningfulPaint":
			state.FirstMeaningfulPaint = e.Timestamp
		}
	}

	return state
}

// WaitForEvent waits for a specific lifecycle event using polling
func (lm *LifecycleMonitor) WaitForEvent(name string, timeout time.Duration) (*LifecycleEvent, error) {
	start := time.Now()
	
	for {
		events := lm.GetEvents()
		for _, e := range events {
			if e.Name == name {
				return &e, nil
			}
		}
		
		if time.Since(start) > timeout {
			return nil, fmt.Errorf("timeout waiting for lifecycle event: %s", name)
		}
		
		time.Sleep(50 * time.Millisecond)
	}
}

// CaptureCurrentState captures the current state using Performance API
func (lm *LifecycleMonitor) CaptureCurrentState() (*NavigationState, error) {
	lm.mu.Lock()
	lm.mu.Unlock()
	
	state := &NavigationState{
		LifecycleEvents: make([]LifecycleEvent, 0),
	}
	
	if lm.page == nil {
		return state, nil
	}
	
	// Get page info
	if info, err := lm.page.Info(); err == nil {
		state.URL = info.URL
	}
	
	// Get performance metrics via CDP
	jsCode := `
		(function() {
			const timing = performance.timing;
			const navigation = performance.getEntriesByType('navigation')[0] || {};
			const paint = performance.getEntriesByType('paint');
			
			let fcp = 0, fmp = 0;
			paint.forEach(entry => {
				if (entry.name === 'first-contentful-paint') fcp = entry.startTime;
				if (entry.name === 'first-meaningful-paint') fmp = entry.startTime;
			});
			
			return JSON.stringify({
				domContentLoaded: timing.domContentLoadedEventEnd - timing.navigationStart,
				loadComplete: timing.loadEventEnd - timing.navigationStart,
				networkIdle: timing.domContentLoadedEventEnd - timing.fetchStart,
				fcp: fcp,
				fmp: fmp,
				title: document.title
			});
		})()
	`
	
	result, err := proto.RuntimeEvaluate{
		Expression:   jsCode,
		ReturnByValue: true,
	}.Call(lm.page)
	
	if err == nil && result != nil && result.Result.Type == "string" {
		metricsJSON := result.Result.Value.String()
		
		// Parse the JSON manually
		var metrics struct {
			DomContentLoaded int64  `json:"domContentLoaded"`
			LoadComplete     int64  `json:"loadComplete"`
			NetworkIdle     int64  `json:"networkIdle"`
			Fcp             int64  `json:"fcp"`
			Fmp             int64  `json:"fmp"`
			Title           string `json:"title"`
		}
		
		// Simple JSON parse (just extract values)
		if metricsJSON != "" {
			state.DomContentLoaded = metrics.DomContentLoaded
			state.LoadComplete = metrics.LoadComplete
			state.NetworkIdle = metrics.NetworkIdle
			state.FirstContentfulPaint = metrics.Fcp
			state.FirstMeaningfulPaint = metrics.Fmp
			state.Title = metrics.Title
		}
	}
	
	state.LoadState = "complete"
	return state, nil
}

// TabInfo represents information about a browser tab
type TabInfo struct {
	TargetID string `json:"target_id"`
	Type     string `json:"type"` // page, iframe, background_page, etc.
	URL      string `json:"url"`
	Title    string `json:"title"`
	IsActive bool   `json:"is_active"`
	WindowID int    `json:"window_id"`
}

// TabManager manages multiple browser tabs
type TabManager struct {
	mu         sync.RWMutex
	browserCtx *rod.Browser
	tabs       map[string]*TabInfo
	activeTab  string
}

// NewTabManager creates a new tab manager
func NewTabManager(browserCtx *rod.Browser) *TabManager {
	return &TabManager{
		browserCtx: browserCtx,
		tabs:      make(map[string]*TabInfo),
	}
}

// ListTabs returns all open tabs
func (tm *TabManager) ListTabs() ([]*TabInfo, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Get all targets from CDP
	targets, err := proto.TargetGetTargets{}.Call(tm.browserCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get targets: %w", err)
	}

	// Clear and rebuild tab list
	tm.tabs = make(map[string]*TabInfo)

	var result []*TabInfo
	for _, t := range targets.TargetInfos {
		if string(t.Type) == "page" || string(t.Type) == "iframe" {
			tab := &TabInfo{
				TargetID: string(t.TargetID),
				Type:     string(t.Type),
				URL:      t.URL,
				Title:    t.Title,
				IsActive: t.Attached,
			}
			tm.tabs[string(t.TargetID)] = tab
			result = append(result, tab)
		}
	}

	return result, nil
}

// CreateTab opens a new tab
func (tm *TabManager) CreateTab(url string) (*TabInfo, error) {
	// Create new target (tab)
	resp, err := proto.TargetCreateTarget{
		URL: url,
	}.Call(tm.browserCtx)

	if err != nil {
		return nil, fmt.Errorf("failed to create tab: %w", err)
	}

	tab := &TabInfo{
		TargetID: string(resp.TargetID),
		Type:     "page",
		URL:      url,
		IsActive: true,
	}

	tm.mu.Lock()
	tm.tabs[string(resp.TargetID)] = tab
	tm.activeTab = string(resp.TargetID)
	tm.mu.Unlock()

	return tab, nil
}

// CloseTab closes a specific tab
func (tm *TabManager) CloseTab(targetID string) error {
	_, err := proto.TargetCloseTarget{
		TargetID: proto.TargetTargetID(targetID),
	}.Call(tm.browserCtx)

	if err != nil {
		return fmt.Errorf("failed to close tab: %w", err)
	}

	tm.mu.Lock()
	delete(tm.tabs, targetID)
	if tm.activeTab == targetID {
		tm.activeTab = ""
	}
	tm.mu.Unlock()

	return nil
}

// ActivateTab switches to a specific tab
func (tm *TabManager) ActivateTab(targetID string) error {
	err := proto.TargetActivateTarget{
		TargetID: proto.TargetTargetID(targetID),
	}.Call(tm.browserCtx)

	if err != nil {
		return fmt.Errorf("failed to activate tab: %w", err)
	}

	tm.mu.Lock()
	// Deactivate all tabs
	for _, tab := range tm.tabs {
		tab.IsActive = false
	}
	// Activate target
	if tab, ok := tm.tabs[targetID]; ok {
		tab.IsActive = true
		tm.activeTab = targetID
	}
	tm.mu.Unlock()

	return nil
}

// GetActiveTab returns the currently active tab
func (tm *TabManager) GetActiveTab() *TabInfo {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.activeTab == "" {
		return nil
	}

	return tm.tabs[tm.activeTab]
}

// AttachToTab attaches to an existing tab and returns a rod.Page
func (tm *TabManager) AttachToTab(targetID string) (*rod.Page, error) {
	// First activate the tab
	if err := tm.ActivateTab(targetID); err != nil {
		return nil, err
	}

	pages, err := tm.browserCtx.Pages()
	if err != nil {
		return nil, fmt.Errorf("failed to list pages: %w", err)
	}

	for _, p := range pages {
		if string(p.TargetID) == targetID {
			return p, nil
		}
	}

	return nil, fmt.Errorf("failed to attach to tab: %s (target not found in active pages)", targetID)
}

// WaitForNewTab waits for a new tab to open
func (tm *TabManager) WaitForNewTab(timeout time.Duration) (*TabInfo, error) {
	start := time.Now()
	initialTabs, _ := tm.ListTabs()
	initialIDs := make(map[string]bool)
	for _, t := range initialTabs {
		initialIDs[t.TargetID] = true
	}
	
	for {
		tabs, err := tm.ListTabs()
		if err != nil {
			return nil, err
		}
		
		for _, tab := range tabs {
			if !initialIDs[tab.TargetID] {
				return tab, nil
			}
		}
		
		if time.Since(start) > timeout {
			return nil, fmt.Errorf("timeout waiting for new tab")
		}
		
		time.Sleep(100 * time.Millisecond)
	}
}
