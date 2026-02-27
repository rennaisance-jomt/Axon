package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rennaisance-jomt/axon/internal/browser"
	"github.com/rennaiseance-jomt/axon/internal/config"
	"github.com/rennaiseance-jomt/axon/internal/security"
	"github.com/rennaiseance-jomt/axon/pkg/types"
)

// Handlers holds all handlers
type Handlers struct {
	sessions    *browser.SessionManager
	pool        *browser.Pool
	ssrfGuard   *security.SSRFGuard
	classifier  *security.ActionClassifier
	auditLogger *security.AuditLogger
	cfg         *config.Config
}

// NewHandlers creates new handlers
func NewHandlers(pool *browser.Pool, cfg *config.Config) *Handlers {
	return &Handlers{
		sessions:    browser.NewSessionManager(pool),
		pool:        pool,
		ssrfGuard:   security.NewSSRFGuard(cfg.Security.SSRF.AllowPrivateNetwork, cfg.Security.SSRF.DomainAllowlist, cfg.Security.SSRF.DomainDenylist, cfg.Security.SSRF.SchemeAllowlist),
		classifier:  security.NewActionClassifier(),
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
	if err := session.Navigate(req.URL); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(types.APIError{
			Error:      true,
			ErrorType:  types.ErrNavigationFailed,
			Message:    err.Error(),
			Suggestion: "Check if the URL is valid and accessible",
			Recoverable: true,
		})
	}

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
	snapshot, err := extractor.Extract(session.Page, req.Depth)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(types.APIError{
			Error:       true,
			ErrorType:   types.ErrTimeout,
			Message:     err.Error(),
			Recoverable: true,
		})
	}

	snapshot.SessionID = id

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
				Message:     err.Errorion:  "(),
				SuggestRun snapshot to get fresh refs",
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

	return c.JSON(types.StatusResponse{
		URL:       session.URL,
		Title:     session.Title,
		AuthState: authState,
		PageState: pageState,
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

	_, err := io.ReadAll(c.Body())
	if err != nil {
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

	_ = session
	return c.SendStatus(http.StatusNoContent)
}

// handleAudit handles audit logs
func (h *Handlers) handleAudit(c *fiber.Ctx) error {
	sessionID := c.Query("session_id")
	limit, _ := strconv.Atoi(c.Query("limit", "100"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	_ = sessionID
	_ = limit
	_ = offset

	// Would retrieve from storage
	return c.JSON(fiber.Map{"logs": []interface{}{}, "total": 0})
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
	h.auditLogger.LogAction(entry)
}

func saveScreenshot(path string, data []byte) error {
	_ = path
	_ = data
	// Simplified - would write to disk
	return nil
}
