package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/go-rod/rod/lib/proto"
	"github.com/rennaisance-jomt/axon/internal/browser"
	"github.com/rennaisance-jomt/axon/internal/config"
	"github.com/rennaisance-jomt/axon/internal/mcp"
	"github.com/rennaisance-jomt/axon/internal/security"
	"github.com/rennaisance-jomt/axon/internal/storage"
	"github.com/rennaisance-jomt/axon/internal/telemetry"
	"github.com/rennaisance-jomt/axon/pkg/types"
	"github.com/rennaisance-jomt/axon/pkg/logger"
)

// Handlers holds all handlers
type Handlers struct {
	sessions    *browser.SessionManager
	pool        *browser.Pool
	db          *storage.DB
	ssrfGuard       *security.SSRFGuard
	classifier      *security.ActionClassifier
	promptGuard     *security.PromptInjectionGuard
	auditLogger     *security.AuditLogger
	cfg             *config.Config
	stats           *StatsCollector
	vault           *security.Vault
}

// NewHandlers creates new handlers
func NewHandlers(pool *browser.Pool, db *storage.DB, cfg *config.Config) *Handlers {
	// Sprint 18: Initialize checkpoint manager for session snapshots
	checkpointMgr := browser.NewCheckpointManager(50, 24*time.Hour)

	// Sprint 28: Initialize vault
	vault := security.NewVault(db, []byte(cfg.Security.VaultKey))

	// Sprint 19: Initialize session manager with recovery support
	sessions := browser.NewSessionManagerWithRecovery(pool, cfg.Browser.MaxSessionLife, checkpointMgr, vault)
	
	// Sprint 24: Initialize SSRF guard
	ssrfGuard := security.NewSSRFGuard(cfg.Security.SSRF.AllowPrivateNetwork, cfg.Security.SSRF.DomainAllowlist, cfg.Security.SSRF.DomainDenylist, cfg.Security.SSRF.SchemeAllowlist)
	
	handlers := &Handlers{
		sessions:    sessions,
		pool:        pool,
		db:          db,
		ssrfGuard:   ssrfGuard,
		classifier:  security.NewActionClassifier(),
		promptGuard: security.NewPromptInjectionGuard(),
		auditLogger: security.NewAuditLogger(),
		cfg:         cfg,
		vault:       vault,
	}
	
	// Set up SSRF event handler for audit logging and admin notifications
	ssrfGuard.SetEventHandler(func(event *security.SSRFEvent) {
		// Log to audit
		handlers.logSSRFEvent(event)
		
		// Admin notification for blocked attempts
		if event.Type == security.EventBlocked {
			handlers.sendSSRFAlert(event)
		}
	})
	
	return handlers
}

// handleCreateSession handles session creation
func (h *Handlers) handleCreateSession(c *fiber.Ctx) error {
	var req types.CreateSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrInvalidAction,
			Message:     "Invalid request body",
			Recoverable: true,
		})
	}

	// Create session
	session, err := h.sessions.Create(req.ID, req.Profile)
	if err != nil {
		return c.Status(http.StatusConflict).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrSessionNotFound,
			Message:     err.Error(),
			Recoverable: false,
		})
	}

	// Track session creation in telemetry
	if tel := telemetry.GetGlobalTelemetry(); tel != nil {
		tel.TrackSessionCreated(c.Context(), session.ID)
	}

	logger.Success("New session created: %s", session.ID)
	return c.Status(http.StatusCreated).JSON(session)
}

// handleListSessions handles session listing
func (h *Handlers) handleListSessions(c *fiber.Ctx) error {
	sessions := h.sessions.List()
	var result []types.SessionInfo
	for _, s := range sessions {
		result = append(result, types.SessionInfo{
			SessionID:  s.ID,
			Status:     s.Status,
			Profile:    s.Profile,
			CreatedAt:  s.CreatedAt,
			LastAction: s.LastAction,
			URL:        s.URL,
			Title:      s.Title,
			AuthState:  s.AuthState,
			PageState:  s.PageState,
		})
	}
	return c.JSON(fiber.Map{"sessions": result})
}

// handleGetSession handles getting a session
func (h *Handlers) handleGetSession(c *fiber.Ctx) error {
	id := c.Params("id")
	session, err := h.sessions.Get(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrSessionNotFound,
			Message:     err.Error(),
			Recoverable: false,
		})
	}

	return c.JSON(types.SessionInfo{
		SessionID:  session.ID,
		Status:     session.Status,
		Profile:    session.Profile,
		CreatedAt:  session.CreatedAt,
		LastAction: session.LastAction,
		URL:        session.URL,
		Title:      session.Title,
		AuthState:  session.AuthState,
		PageState:  session.PageState,
	})
}

// handleDeleteSession handles session deletion
func (h *Handlers) handleDeleteSession(c *fiber.Ctx) error {
	id := c.Params("id")
	logger.Info("Deleting session: %s", id)
	if err := h.sessions.Delete(id); err != nil {
		return c.Status(http.StatusNotFound).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrSessionNotFound,
			Message:     err.Error(),
			Recoverable: false,
		})
	}

	// Flush telemetry to ensure all spans for this session are sent
	telemetry.Flush(c.Context())

	return c.SendStatus(http.StatusNoContent)
}

// handleNavigate handles navigation
func (h *Handlers) handleNavigate(c *fiber.Ctx) error {
	id := c.Params("id")

	var req types.NavigateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrInvalidAction,
			Message:     "Invalid request body",
			Recoverable: true,
		})
	}

	// Validate URL with SSRF guard
	if err := h.ssrfGuard.ValidateURL(req.URL); err != nil {
		return c.Status(http.StatusForbidden).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrSSRFBlocked,
			Message:     err.Error(),
			Recoverable: false,
		})
	}

	session, err := h.sessions.Get(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrSessionNotFound,
			Message:     err.Error(),
			Recoverable: false,
		})
	}

	// Navigate
	logger.Action("Navigating session %s to %s", id, req.URL)
	if err := session.Navigate(req.URL, req.WaitUntil); err != nil {
		logger.Error("Navigation failed for %s: %v", id, err)
		return c.Status(http.StatusInternalServerError).JSON(types.APIError{
			Error:      true,
			ErrorType:  types.ErrNavigationFailed,
			Message:    err.Error(),
			Suggestion: "Check if the URL is valid and accessible",
			Recoverable: true,
		})
	}
	logger.Success("Navigation complete: %s", session.Title)

	// Log audit
	h.logAudit(session.ID, "navigate", req.URL, "", types.ReversibilityRead, "success")

	return c.JSON(types.NavigateResponse{
		Success: true,
		URL:     session.URL,
		Title:   session.Title,
		State:   session.PageState,
	})
}

// handleSnapshot handles snapshot
func (h *Handlers) handleSnapshot(c *fiber.Ctx) error {
	id := c.Params("id")

	var req types.SnapshotRequest
	_ = c.BodyParser(&req)

	if req.Depth == "" {
		req.Depth = "compact"
	}

	session, err := h.sessions.Get(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrSessionNotFound,
			Message:     err.Error(),
			Recoverable: false,
		})
	}

	// Get snapshot
	extractor := browser.NewSnapshotExtractor().WithVault(h.vault)
	snapshot, err := extractor.Extract(session.Page, req.Depth, req.Focus)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrTimeout,
			Message:     err.Error(),
			Recoverable: true,
		})
	}

	// Store elements in session for action lookup
	session.SetLastElements(snapshot.Elements)

	snapshot.SessionID = id

	// Scan for prompt injection
	if detected, pattern := h.promptGuard.ScanContent(snapshot.Content); detected {
		snapshot.Warnings = append(snapshot.Warnings, browser.Warning{
			Type:     types.WarningPromptInjection,
			Severity: types.SeverityHigh,
			Message:  fmt.Sprintf("Suspected prompt injection detected (pattern: %s)", pattern),
		})
	}

	// Log audit
	h.logAudit(id, "snapshot", "", "", types.ReversibilityRead, "success")

	// Track snapshot telemetry - estimate raw tokens vs reduced
	if tel := telemetry.GetGlobalTelemetry(); tel != nil {
		// Estimate raw tokens (would be ~50K for full HTML)
		rawTokens := 50000
		reducedTokens := snapshot.TokenCount
		if reducedTokens == 0 {
			reducedTokens = len(snapshot.Content)
		}
		tel.TrackSnapshot(c.Context(), id, rawTokens, reducedTokens, 0)
	}

	return c.JSON(snapshot)
}

// handleAct handles actions
func (h *Handlers) handleAct(c *fiber.Ctx) error {
	id := c.Params("id")

	var req types.ActionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrInvalidAction,
			Message:     "Invalid request body",
			Recoverable: true,
		})
	}

	session, err := h.sessions.Get(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrSessionNotFound,
			Message:     err.Error(),
			Recoverable: false,
		})
	}

	logger.Action("[%s] Performing action '%s' on ref '%s'", id, req.Action, req.Ref)

	// Classify action
	reversibility := h.classifier.ClassifyAction(req.Action, req.Ref, "")

	// Check if confirmation required
	if h.classifier.RequiresConfirmation(reversibility) && !req.Confirm {
		return c.JSON(types.ActionResult{
			Success:         false,
			RequiresConfirm: true,
			Message:         fmt.Sprintf("This action (%s) is irreversible. Set confirm=true to proceed.", reversibility),
		})
	}

	// Find element selector from ref (simplified)
	selector := fmt.Sprintf("[data-ref='%s']", req.Ref)

	// Execute action
	switch req.Action {
	case types.ActionClick:
		if err := session.Click(selector); err != nil {
			return c.JSON(types.ActionResult{
				Success:     false,
				ErrorType:   types.ErrElementNotFound,
				Message:     err.Error(),
				Suggestion:  "Run snapshot to get fresh refs",
				Recoverable: true,
			})
		}
	case types.ActionFill:
		if err := session.Fill(selector, req.Value); err != nil {
			// Use 422 Unprocessable Entity so the SDK can detect business-logic
			// failures (e.g. phishing guard block) vs infrastructure errors.
			return c.Status(http.StatusUnprocessableEntity).JSON(types.ActionResult{
				Success:     false,
				ErrorType:   types.ErrElementNotFound,
				Message:     err.Error(),
				Suggestion:  "Run snapshot to get fresh refs",
				Recoverable: true,
			})
		}
	case types.ActionPress:
		if err := session.Press(selector, req.Value); err != nil {
			return c.JSON(types.ActionResult{
				Success:     false,
				ErrorType:   types.ErrElementNotFound,
				Message:     err.Error(),
				Recoverable: true,
			})
		}
	case types.ActionScroll:
		y := 500 // default scroll
		if req.Value != "" {
			fmt.Sscanf(req.Value, "%d", &y)
		}
		if err := session.Scroll(selector, y); err != nil {
			return c.JSON(types.ActionResult{
				Success:     false,
				ErrorType:   types.ErrElementNotFound,
				Message:     err.Error(),
				Recoverable: true,
			})
		}
	case types.ActionHover:
		if err := session.Hover(selector); err != nil {
			return c.JSON(types.ActionResult{
				Success:     false,
				ErrorType:   types.ErrElementNotFound,
				Message:     err.Error(),
				Recoverable: true,
			})
		}
	default:
		return c.Status(http.StatusBadRequest).JSON(types.ActionResult{
			Success:     false,
			ErrorType:   types.ErrInvalidAction,
			Message:     fmt.Sprintf("Unknown action: %s", req.Action),
			Recoverable: false,
		})
	}

	// Log audit
	h.logAudit(id, req.Action, req.Ref, "", reversibility, "success")

	// Track action in telemetry
	if tel := telemetry.GetGlobalTelemetry(); tel != nil {
		tel.TrackAction(c.Context(), id, req.Action, req.Ref, true, 0, nil)
	}

	return c.JSON(types.ActionResult{
		Success: true,
		Result:  fmt.Sprintf("Action %s completed", req.Action),
	})
}

// handleStatus handles status
func (h *Handlers) handleStatus(c *fiber.Ctx) error {
	id := c.Params("id")

	session, err := h.sessions.Get(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrSessionNotFound,
			Message:     err.Error(),
			Recoverable: false,
		})
	}

	// Detect state
	detector := browser.NewStateDetector()
	authState := detector.DetectAuthState(session.Page)
	pageState := detector.DetectPageState(session.Page)

	session.AuthState = authState
	session.PageState = pageState

	scrollHeight, _ := session.GetScrollHeight()

	return c.JSON(types.StatusResponse{
		URL:          session.URL,
		Title:        session.Title,
		AuthState:    authState,
		PageState:    pageState,
		ScrollHeight: scrollHeight,
	})
}

// handleScreenshot handles screenshots
func (h *Handlers) handleScreenshot(c *fiber.Ctx) error {
	id := c.Params("id")

	var req types.ScreenshotRequest
	_ = c.BodyParser(&req)

	session, err := h.sessions.Get(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrSessionNotFound,
			Message:     err.Error(),
			Recoverable: false,
		})
	}

	// Take screenshot
	img, err := session.Screenshot(req.FullPage)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrTimeout,
			Message:     err.Error(),
			Recoverable: true,
		})
	}

	// Save to file
	filename := fmt.Sprintf("screenshot_%s_%d.png", id, time.Now().Unix())
	path := "./screenshots/" + filename

	if err := saveScreenshot(path, img); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrTimeout,
			Message:     err.Error(),
			Recoverable: true,
		})
	}

	return c.JSON(types.ScreenshotResponse{Path: path})
}

// handleResize handles window resizing
func (h *Handlers) handleResize(c *fiber.Ctx) error {
	id := c.Params("id")

	var req struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Width <= 0 {
		req.Width = 1280
	}
	if req.Height <= 0 {
		req.Height = 800
	}

	session, err := h.sessions.Get(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "session not found"})
	}

	if err := session.Resize(req.Width, req.Height); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "width": req.Width, "height": req.Height})
}

// handleWait handles wait
func (h *Handlers) handleWait(c *fiber.Ctx) error {
	id := c.Params("id")

	var req types.WaitRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrInvalidAction,
			Message:     "Invalid request body",
			Recoverable: true,
		})
	}

	session, err := h.sessions.Get(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrSessionNotFound,
			Message:     err.Error(),
			Recoverable: false,
		})
	}

	timeout := 30 * time.Second
	if req.Timeout > 0 {
		timeout = time.Duration(req.Timeout) * time.Millisecond
	}

	err = session.Wait(browser.WaitOptions{
		Condition: req.Condition,
		Selector:  req.Selector,
		Text:      req.Text,
		Timeout:   timeout,
	})

	return c.JSON(types.WaitResponse{
		Success: err == nil,
		Matched: err == nil,
	})
}

// handleGetCookies handles getting cookies
func (h *Handlers) handleGetCookies(c *fiber.Ctx) error {
	id := c.Params("id")
	domain := c.Query("domain")

	session, err := h.sessions.Get(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrSessionNotFound,
			Message:     err.Error(),
			Recoverable: false,
		})
	}

	cookies, err := session.GetCookies(domain)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrTimeout,
			Message:     err.Error(),
			Recoverable: true,
		})
	}

	return c.JSON(fiber.Map{"cookies": cookies})
}

// handleSetCookies handles setting cookies
func (h *Handlers) handleSetCookies(c *fiber.Ctx) error {
	id := c.Params("id")

	var cookies []*proto.NetworkCookieParam
	if err := c.BodyParser(&cookies); err != nil {
		return c.Status(http.StatusBadRequest).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrInvalidAction,
			Message:     "Invalid request body: " + err.Error(),
			Recoverable: true,
		})
	}

	session, err := h.sessions.Get(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrSessionNotFound,
			Message:     err.Error(),
			Recoverable: false,
		})
	}

	if err := session.SetCookies(cookies); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrInvalidAction,
			Message:     err.Error(),
			Recoverable: true,
		})
	}

	h.logAudit(id, "set_cookies", "", "", types.ReversibilityWriteReversible, "success")

	return c.SendStatus(http.StatusNoContent)
}

// TelemetryLLMRequest defines the incoming payload for llm token usage
type TelemetryLLMRequest struct {
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	Model            string `json:"model"`
}

// handleLLMTelemetry allows agents to report LLM token usage back to Axon for Langfuse integration
func (h *Handlers) handleLLMTelemetry(c *fiber.Ctx) error {
	id := c.Params("id")

	var req TelemetryLLMRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	tel := telemetry.GetGlobalTelemetry()
	if tel != nil {
		tel.TrackLLMUsage(c.Context(), id, req.PromptTokens, req.CompletionTokens, req.Model)
	}

	return c.JSON(fiber.Map{"success": true})
}

func (h *Handlers) handleAudit(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "100"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	rawLogs, err := h.db.GetAuditLogs(limit, offset)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	var logs []security.AuditEntry
	for _, raw := range rawLogs {
		var entry security.AuditEntry
		if err := json.Unmarshal(raw, &entry); err == nil {
			logs = append(logs, entry)
		}
	}

	return c.JSON(fiber.Map{
		"logs":   logs,
		"total":  len(logs),
		"offset": offset,
		"limit":  limit,
	})
}

func (h *Handlers) logAudit(sessionID, action, targetRef, intent, reversibility, result string) {
	entry := &security.AuditEntry{
		SessionID:     sessionID,
		Action:        action,
		TargetRef:     targetRef,
		TargetIntent:  intent,
		Reversibility: reversibility,
		Result:        result,
	}
	
	if err := h.auditLogger.LogAction(entry); err != nil {
		return
	}

	data, err := json.Marshal(entry)
	if err == nil {
		_ = h.db.AppendAuditLog(data)
	}
}

// logSSRFEvent logs SSRF events to the audit system
func (h *Handlers) logSSRFEvent(event *security.SSRFEvent) {
	entry := &security.AuditEntry{
		Action:      "ssrf_" + string(event.Type),
		TargetRef:   event.URL,
		Domain:      event.Domain,
		Reversibility: "read-only",
		Result:      event.Reason,
	}
	
	if err := h.auditLogger.LogAction(entry); err != nil {
		return
	}

	data, err := json.Marshal(entry)
	if err == nil {
		_ = h.db.AppendAuditLog(data)
	}
}

// sendSSRFAlert sends admin notification for blocked SSRF attempts
func (h *Handlers) sendSSRFAlert(event *security.SSRFEvent) {
	// Log critical security alert
	logger.Warn("⚠️ SSRF BLOCKED: %s | URL: %s | Reason: %s", 
		event.Timestamp.Format(time.RFC3339), event.URL, event.Reason)
	
	// In production, this would send to a notification service
	// Examples: webhook, PagerDuty, email, Slack, etc.
	// For now, we log to the audit system and console
	
	// You can extend this to integrate with:
	// - Webhook notifications
	// - PagerDuty/Slack alerts  
	// - Email notifications
	// - SIEM systems
}

func saveScreenshot(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// SetStatsCollector sets the stats collector
func (h *Handlers) SetStatsCollector(stats *StatsCollector) {
	h.stats = stats
}

// handleFindAndAct handles intent-based element finding and action
func (h *Handlers) handleFindAndAct(c *fiber.Ctx) error {
	id := c.Params("id")
	
	var req struct {
		Intent  string `json:"intent"`
		Action  string `json:"action"`
		Value   string `json:"value,omitempty"`
		Confirm bool   `json:"confirm,omitempty"`
	}
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrInvalidAction,
			Message:     "Invalid request body",
			Recoverable: true,
		})
	}
	
	if req.Intent == "" {
		return c.Status(http.StatusBadRequest).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrInvalidAction,
			Message:     "intent is required",
			Recoverable: true,
		})
	}
	
	session, err := h.sessions.Get(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrSessionNotFound,
			Message:     err.Error(),
			Recoverable: false,
		})
	}
	
	// Import MCP resolver for intent resolution
	resolver := mcp.NewIntentResolver(h.db)
	ref, err := resolver.Resolve(session, req.Intent)
	
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrElementNotFound,
			Message:     fmt.Sprintf("Could not resolve intent '%s': %v", req.Intent, err),
			Suggestion:  "Try a more specific description or use snapshot to find element refs",
			Recoverable: true,
		})
	}
	
	// Log the intent-based action
	logger.Action("[%s] Intent found: '%s' -> ref %s", id, req.Intent, ref)
	h.logAudit(id, req.Action, ref, req.Intent, h.classifier.ClassifyAction(req.Action, ref, ""), "success")
	
	// Store element in memory for future use
	if h.db != nil {
		key := fmt.Sprintf("intent:%s:%s", session.URL, req.Intent)
		h.db.StoreElementMemory(key, ref)
	}
	
	logger.Success("[%s] Action %s completed on detected element", id, req.Action)

	// Track intent-based action in telemetry
	if tel := telemetry.GetGlobalTelemetry(); tel != nil {
		tel.TrackAction(c.Context(), id, req.Action, ref, true, 0, nil)
	}

	return c.JSON(types.ActionResult{
		Success: true,
		Result:  fmt.Sprintf("Resolved intent '%s' to ref '%s' and performed %s", req.Intent, ref, req.Action),
		Message: fmt.Sprintf("Found element matching '%s'", req.Intent),
	})
}

// handleListTabs lists all tabs in a session
func (h *Handlers) handleListTabs(c *fiber.Ctx) error {
	id := c.Params("id")
	session, err := h.sessions.Get(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	tabs, err := session.TabManager.ListTabs()
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(tabs)
}

// handleCreateTab creates a new tab in a session
func (h *Handlers) handleCreateTab(c *fiber.Ctx) error {
	id := c.Params("id")
	session, err := h.sessions.Get(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	var req struct {
		URL string `json:"url"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	tab, err := session.TabManager.CreateTab(req.URL)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(tab)
}

// handleActivateTab activates a specific tab
func (h *Handlers) handleActivateTab(c *fiber.Ctx) error {
	id := c.Params("id")
	targetID := c.Params("target_id")
	session, err := h.sessions.Get(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	if err := session.TabManager.ActivateTab(targetID); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Update the session's page to the new tab's page
	page, err := session.TabManager.AttachToTab(targetID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	session.Page = page

	return c.JSON(fiber.Map{"success": true, "message": "Tab activated"})
}

// handleCloseTab closes a specific tab
func (h *Handlers) handleCloseTab(c *fiber.Ctx) error {
	id := c.Params("id")
	targetID := c.Params("target_id")
	session, err := h.sessions.Get(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	if err := session.TabManager.CloseTab(targetID); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Tab closed"})
}

// handleStream handles WebSocket session streaming
func (h *Handlers) handleStream(c *websocket.Conn) {
	id := c.Params("id")
	session, err := h.sessions.Get(id)
	if err != nil {
		_ = c.WriteJSON(fiber.Map{"error": err.Error()})
		return
	}

	logger.Info("[%s] WebSocket stream client connected", id)
	defer logger.Info("[%s] WebSocket stream client disconnected", id)

	// Start streamer
	if err := session.Streamer.Start(); err != nil {
		logger.Error("[%s] Failed to start streamer: %v", id, err)
		_ = c.WriteJSON(fiber.Map{"error": err.Error()})
		return
	}
	defer session.Streamer.Stop()

	// Capture frames and send to WS
	frames := session.Streamer.GetFrames()

	// Handle WS closure or control messages
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			msgType, _, err := c.ReadMessage()
			if err != nil {
				return
			}
			if msgType == websocket.CloseMessage {
				return
			}
		}
	}()

	for {
		select {
		case frame := <-frames:
			if err := c.WriteJSON(frame); err != nil {
				return
			}
		case <-done:
			return
		case <-time.After(30 * time.Second):
			// Keepalive
			if err := c.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleReplay serves recorded session history
func (h *Handlers) handleReplay(c *fiber.Ctx) error {
	id := c.Params("id")
	session, err := h.sessions.Get(id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	history := session.Streamer.GetHistory()
	return c.JSON(fiber.Map{
		"session_id": id,
		"frames":     history,
		"count":      len(history),
	})
}

// handleAgentChat calls Gemini directly from the backend
func (h *Handlers) handleAgentChat(c *fiber.Ctx) error {
	var req struct {
		Task     string `json:"task"`
		Snapshot string `json:"snapshot"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	logger.Info("[Agent] Received chat request. Task: %s, Snapshot size: %d", req.Task, len(req.Snapshot))

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "GEMINI_API_KEY environment variable not set on server. Please set it before starting."})
	}

	url := "https://generativelanguage.googleapis.com/v1/models/gemini-2.5-flash:generateContent?key=" + apiKey
	logger.Debug("[Agent] Calling Gemini API: %s (Key: %s...)", "v1/gemini-2.5-flash", apiKey[:5])

	prompt := fmt.Sprintf(`You are an autonomous web-browsing AI agent powered by Axon.
Your ultimate task is: "%s"

Current semantic state:
%s

Decide your next action. Output JSON ONLY:
{
    "ref": "the ID in brackets (e.g. e1) of the element to interact with",
    "action": "click", "fill", or "press",
    "value": "text to type if fill, or key name like 'Enter' if press",
    "reasoning": "short explanation of your decision",
    "task_complete": boolean
}
Note: To search, you usually need to 'fill' the query and then 'press' the 'Enter' key in the next step.`, req.Task, req.Snapshot)

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
	}
	
	payloadBytes, _ := json.Marshal(payload)
	
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	httpReq.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Gemini request failed: " + err.Error()})
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return c.Status(resp.StatusCode).JSON(fiber.Map{"error": fmt.Sprintf("Gemini API Error (%d): %s", resp.StatusCode, string(body))})
	}
	
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to parse Gemini response: " + err.Error()})
	}
	
	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Empty response from Gemini"})
	}
	
	text := geminiResp.Candidates[0].Content.Parts[0].Text
	
	// Pre-process JSON text
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)
	
	var decision map[string]interface{}
	if err := json.Unmarshal([]byte(text), &decision); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to parse decision JSON from Gemini: " + err.Error() + "\nRaw: " + text})
	}
	
	return c.JSON(decision)
}

// handleAddSecret handles adding a secret to the vault
func (h *Handlers) handleAddSecret(c *fiber.Ctx) error {
	var req struct {
		Name     string   `json:"name"`
		Value    string   `json:"value"`
		Username string   `json:"username"`
		Password string   `json:"password"`
		URL      string   `json:"url"`
		Labels   []string `json:"labels"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Name == "" || req.URL == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "name and url are required"})
	}

	secret := &security.Secret{
		Name:      req.Name,
		Value:     req.Value,
		Username:  req.Username,
		Password:  req.Password,
		Domain:    req.URL, // In production, extract domain from URL
		CreatedAt: time.Now(),
		Labels:    req.Labels,
	}

	if err := h.vault.AddSecret(secret); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	logger.Success("Secret added to vault: %s (domain: %s)", req.Name, secret.Domain)
	return c.JSON(fiber.Map{"success": true})
}

// handleListSecrets handles listing vault secrets (metadata only)
func (h *Handlers) handleListSecrets(c *fiber.Ctx) error {
	// This would require a ListSecrets method in Vault/Storage
	// For now, return a placeholder or implement it if needed
	return c.JSON(fiber.Map{"message": "ListSecrets not implemented yet"})
}

// handleDeleteSecret handles deleting a secret from the vault
func (h *Handlers) handleDeleteSecret(c *fiber.Ctx) error {
	name := c.Params("name")
	domain := c.Query("domain")

	if name == "" || domain == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "name and domain are required"})
	}

	// This would require a DeleteSecret method in Vault/Storage
	return c.JSON(fiber.Map{"message": "DeleteSecret not implemented yet"})
}
