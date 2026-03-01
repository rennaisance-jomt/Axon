package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/go-rod/rod/lib/proto"
	"github.com/rennaisance-jomt/axon/internal/browser"
	"github.com/rennaisance-jomt/axon/internal/config"
	"github.com/rennaisance-jomt/axon/internal/mcp"
	"github.com/rennaisance-jomt/axon/internal/security"
	"github.com/rennaisance-jomt/axon/internal/storage"
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
}

// NewHandlers creates new handlers
func NewHandlers(pool *browser.Pool, db *storage.DB, cfg *config.Config) *Handlers {
	return &Handlers{
		sessions:    browser.NewSessionManager(pool, cfg.Browser.MaxSessionLife),
		pool:        pool,
		db:          db,
		ssrfGuard:   security.NewSSRFGuard(cfg.Security.SSRF.AllowPrivateNetwork, cfg.Security.SSRF.DomainAllowlist, cfg.Security.SSRF.DomainDenylist, cfg.Security.SSRF.SchemeAllowlist),
		classifier:  security.NewActionClassifier(),
		promptGuard: security.NewPromptInjectionGuard(),
		auditLogger: security.NewAuditLogger(),
		cfg:         cfg,
	}
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
	extractor := browser.NewSnapshotExtractor()
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
			return c.JSON(types.ActionResult{
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

	return c.JSON(types.ActionResult{
		Success: true,
		Result:  fmt.Sprintf("Resolved intent '%s' to ref '%s' and performed %s", req.Intent, ref, req.Action),
		Message: fmt.Sprintf("Found element matching '%s'", req.Intent),
	})
}
