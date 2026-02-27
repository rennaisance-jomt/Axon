package types

import "time"

// Error codes for structured errors
const (
	ErrElementNotFound       = "element_not_found"
	ErrNavigationFailed      = "navigation_failed"
	ErrTimeout               = "timeout"
	ErrCaptcha               = "captcha"
	ErrRateLimited           = "rate_limited"
	ErrAuthRequired          = "auth_required"
	ErrSSRFBlocked           = "ssrf_blocked"
	ErrIrreversibleUnconfirmed = "irreversible_unconfirmed"
	ErrInjectionWarning      = "injection_warning"
	ErrSessionNotFound       = "session_not_found"
	ErrInvalidAction         = "invalid_action"
	ErrInvalidURL            = "invalid_url"
)

// Action types
const (
	ActionClick     = "click"
	ActionFill      = "fill"
	ActionPress     = "press"
	ActionSelect    = "select"
	ActionHover     = "hover"
	ActionScroll    = "scroll"
	ActionNavigate  = "navigate"
	ActionSnapshot  = "snapshot"
	ActionScreenshot = "screenshot"
)

// Wait conditions
const (
	WaitLoad        = "load"
	WaitNetworkIdle = "networkidle"
	WaitDOMContent  = "domcontentloaded"
)

// Reversibility levels
const (
	ReversibilityRead              = "read"
	ReversibilityWriteReversible   = "write_reversible"
	ReversibilityWriteIrreversible = "write_irreversible"
	ReversibilitySensitiveWrite    = "sensitive_write"
)

// Session states
const (
	SessionStatusCreated  = "created"
	SessionStatusActive  = "active"
	SessionStatusIdle    = "idle"
	SessionStatusClosed  = "closed"
)

// Auth states
const (
	AuthStateUnknown   = "unknown"
	AuthStateLoggedOut = "logged_out"
	AuthStateLoggedIn  = "logged_in"
	AuthStateError     = "error"
)

// Page states
const (
	PageStateUnknown   = "unknown"
	PageStateLoading   = "loading"
	PageStateReady     = "ready"
	PageStateError     = "error"
	PageStateCaptcha   = "captcha"
	PageStateRateLimited = "rate_limited"
)

// Warning types
const (
	WarningPromptInjection = "prompt_injection_suspected"
	WarningSSRFAttempt    = "ssrf_attempt"
	WarningUntrusted      = "untrusted_content"
	WarningIrreversible   = "irreversible_action"
	WarningRateLimit      = "rate_limit_warning"
	WarningCaptcha        = "captcha_detected"
	WarningAuthExpired    = "auth_expired"
)

// Severity levels
const (
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

// APIError represents a structured API error
type APIError struct {
	Error       bool     `json:"error"`
	ErrorType   string   `json:"error_type"`
	Message     string   `json:"message"`
	Suggestion  string   `json:"suggestion,omitempty"`
	Recoverable bool     `json:"recoverable"`
}

// ActionRequest represents an action request
type ActionRequest struct {
	Ref     string `json:"ref"`
	Action  string `json:"action"`
	Value   string `json:"value,omitempty"`
	Confirm bool   `json:"confirm"`
}

// ActionResult represents an action result
type ActionResult struct {
	Success         bool   `json:"success"`
	Result          string `json:"result,omitempty"`
	RequiresConfirm bool   `json:"requires_confirm,omitempty"`
	Message         string `json:"message,omitempty"`
	ErrorType       string `json:"error_type,omitempty"`
	Suggestion      string `json:"suggestion,omitempty"`
	Recoverable     bool   `json:"recoverable"`
}

// NavigateRequest represents a navigation request
type NavigateRequest struct {
	URL       string `json:"url"`
	WaitUntil string `json:"wait_until,omitempty"`
}

// NavigateResponse represents a navigation response
type NavigateResponse struct {
	Success bool   `json:"success"`
	URL     string `json:"url"`
	Title   string `json:"title"`
	State   string `json:"state"`
}

// SnapshotRequest represents a snapshot request
type SnapshotRequest struct {
	Focus string `json:"focus,omitempty"`
	Depth string `json:"depth,omitempty"`
}

// SessionInfo represents session info
type SessionInfo struct {
	SessionID   string    `json:"session_id"`
	Status      string    `json:"status"`
	Profile     string    `json:"profile,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	LastAction  time.Time `json:"last_action,omitempty"`
	URL         string    `json:"url,omitempty"`
	Title       string    `json:"title,omitempty"`
	AuthState   string    `json:"auth_state,omitempty"`
	PageState   string    `json:"page_state,omitempty"`
}

// CreateSessionRequest represents a create session request
type CreateSessionRequest struct {
	ID       string `json:"id"`
	Profile  string `json:"profile,omitempty"`
	Headless *bool  `json:"headless,omitempty"`
}

// StatusResponse represents a status response
type StatusResponse struct {
	URL        string `json:"url"`
	Title      string `json:"title"`
	AuthState  string `json:"auth_state"`
	PageState  string `json:"page_state"`
	Warnings   []Warning `json:"warnings"`
}

// Warning represents a warning
type Warning struct {
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// ScreenshotRequest represents a screenshot request
type ScreenshotRequest struct {
	FullPage bool   `json:"full_page,omitempty"`
	Ref      string `json:"ref,omitempty"`
}

// ScreenshotResponse represents a screenshot response
type ScreenshotResponse struct {
	Path string `json:"path"`
}

// WaitRequest represents a wait request
type WaitRequest struct {
	Condition string `json:"condition"`
	Selector  string `json:"selector,omitempty"`
	Text      string `json:"text,omitempty"`
	Timeout   int    `json:"timeout,omitempty"`
}

// WaitResponse represents a wait response
type WaitResponse struct {
	Success bool `json:"success"`
	Matched bool `json:"matched"`
}

// AuditLogEntry represents an audit log entry
type AuditLogEntry struct {
	ID            string                 `json:"id"`
	Timestamp     time.Time             `json:"timestamp"`
	SessionID     string                 `json:"session_id"`
	AgentID       string                 `json:"agent_id,omitempty"`
	Action        string                 `json:"action"`
	TargetRef     string                 `json:"target_ref,omitempty"`
	TargetIntent  string                 `json:"target_intent,omitempty"`
	Domain        string                 `json:"domain,omitempty"`
	Reversibility string                 `json:"reversibility"`
	ConfirmedBy   string                 `json:"confirmed_by,omitempty"`
	Parameters    map[string]interface{} `json:"parameters,omitempty"`
	Result        string                 `json:"result"`
	PrevHash      string                 `json:"prev_hash"`
	ThisHash      string                 `json:"this_hash"`
}
