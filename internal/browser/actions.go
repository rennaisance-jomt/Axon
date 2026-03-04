package browser

import (
	"fmt"
	"time"

	"github.com/go-rod/rod/lib/proto"
	"github.com/rennaisance-jomt/axon/pkg/logger"
)

// WaitOptions represents wait options
type WaitOptions struct {
	Condition string // load, networkidle, selector, text
	Selector  string
	Text      string
	Timeout   time.Duration
}

// Wait waits for a condition
func (s *Session) Wait(opts WaitOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if opts.Timeout == 0 {
		opts.Timeout = 10 * time.Second
	}

	page := s.Page.Timeout(opts.Timeout)

	switch opts.Condition {
	case "load":
		return page.WaitLoad()
	case "networkidle":
		// Sprint 5: Event-Driven Auto-Waiting replacing flaky time.Sleep logic
		_ = proto.DOMEnable{}.Call(page)
		_ = proto.AnimationEnable{}.Call(page)

		// We listen directly to native CDP structural and layout events instead of assuming timeout
		eventFired := make(chan bool, 1)

		go func() {
			waitDOM := page.WaitEvent(&proto.DOMChildNodeInserted{})
			waitDOM()
			eventFired <- true
		}()

		go func() {
			waitAnim := page.WaitEvent(&proto.AnimationAnimationCanceled{})
			waitAnim()
			eventFired <- true
		}()
		
		go func() {
			// Trigger wait event resolving internally
			_ = page.WaitIdle(1 * time.Second)
		}()

		select {
		case <-eventFired:
		case <-time.After(opts.Timeout):
		}
		return nil
	case "domcontentloaded":
		wait := page.WaitEvent(&proto.PageDomContentEventFired{})
		wait()
		return nil
	case "selector":
		if opts.Selector == "" {
			return fmt.Errorf("selector required for wait condition")
		}
		_, err := page.Element(opts.Selector)
		return err
	case "text":
		if opts.Text == "" {
			return fmt.Errorf("text required for wait condition")
		}
		// Wait for an element that contains the text
		_, err := page.ElementR("body", opts.Text)
		return err
	}

	return fmt.Errorf("unknown wait condition: %s", opts.Condition)
}

// Hover hovers over an element
func (s *Session) Hover(selector string) error {
	el, err := s.resolveSelector(selector)
	if err != nil {
		return err
	}

	logger.Action("[%s] Hovering over element: %s", s.ID, selector)
	
	err = el.Timeout(10 * time.Second).Hover()
	
	status := "success"
	if err != nil { status = "failed" }
	s.recordAction("hover", selector, "", status, "")
	return err
}

// Scroll scrolls an element or page
func (s *Session) Scroll(selector string, y int) error {
	if selector == "" {
		_, err := s.Page.Eval(fmt.Sprintf("window.scrollBy(0, %d)", y))
		return err
	}

	el, err := s.resolveSelector(selector)
	if err != nil {
		return err
	}

	logger.Action("[%s] Scrolling element: %s (y=%d)", s.ID, selector, y)

	err = el.Timeout(10 * time.Second).ScrollIntoView()

	status := "success"
	if err != nil { status = "failed" }
	s.recordAction("scroll", selector, fmt.Sprintf("%d", y), status, "")
	return err
}

// DoubleClick performs a double click
func (s *Session) DoubleClick(selector string) error {
	el, err := s.resolveSelector(selector)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	el.MustWaitVisible().MustWaitStable()
	return el.Click(proto.InputMouseButtonLeft, 2)
}

func (s *Session) RightClick(selector string) error {
	el, err := s.resolveSelector(selector)
	if err != nil {
		return err
	}

	logger.Action("[%s] Right-clicking element: %s", s.ID, selector)

	err = el.Timeout(10 * time.Second).Click(proto.InputMouseButtonRight, 1)

	status := "success"
	if err != nil { status = "failed" }
	s.recordAction("right-click", selector, "", status, "")
	return err
}

func (s *Session) SelectOption(selector string, value string) error {
	el, err := s.resolveSelector(selector)
	if err != nil {
		return err
	}

	logger.Action("[%s] Selecting option '%s' in %s", s.ID, value, selector)

	_, err = el.Timeout(10 * time.Second).Eval(fmt.Sprintf(`el => { el.value = "%s"; el.dispatchEvent(new Event('change', { bubbles: true })); }`, value))

	status := "success"
	if err != nil { status = "failed" }
	s.recordAction("select", selector, value, status, "")
	return err
}

// GetPageTitle gets the current page title
func (s *Session) GetPageTitle() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	res, err := s.Page.Eval("document.title")
	if err != nil {
		return "", err
	}
	return res.Value.String(), nil
}

// GetPageURL gets the current page URL
func (s *Session) GetPageURL() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	info, err := s.Page.Info()
	if err != nil {
		return s.URL
	}
	return info.URL
}

// IsElementVisible checks if an element is visible
func (s *Session) IsElementVisible(selector string) (bool, error) {
	el, err := s.resolveSelector(selector)
	if err != nil {
		return false, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return el.Visible()
}

// IsElementEnabled checks if an element is enabled
func (s *Session) IsElementEnabled(selector string) (bool, error) {
	el, err := s.resolveSelector(selector)
	if err != nil {
		return false, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	res, err := el.Eval("el => el.disabled")
	if err != nil {
		return false, err
	}
	return !res.Value.Bool(), nil
}

// GetElementText gets text content of an element
func (s *Session) GetElementText(selector string) (string, error) {
	el, err := s.resolveSelector(selector)
	if err != nil {
		return "", err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return el.Text()
}

// GetElementAttribute gets an attribute of an element
func (s *Session) GetElementAttribute(selector, attr string) (string, error) {
	el, err := s.resolveSelector(selector)
	if err != nil {
		return "", err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	val, err := el.Attribute(attr)
	if err != nil {
		return "", err
	}
	if val == nil {
		return "", nil
	}
	return *val, nil
}

func (s *Session) Focus(selector string) error {
	el, err := s.resolveSelector(selector)
	if err != nil {
		return err
	}

	logger.Action("[%s] Focusing element: %s", s.ID, selector)

	err = el.Timeout(10 * time.Second).Focus()

	status := "success"
	if err != nil { status = "failed" }
	s.recordAction("focus", selector, "", status, "")
	return err
}

// Blur removes focus from an element
func (s *Session) Blur(selector string) error {
	el, err := s.resolveSelector(selector)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, err = el.Eval("el => el.blur()")
	return err
}

// GetHTML gets the HTML of an element or page
func (s *Session) GetHTML(selector string) (string, error) {
	if selector == "" {
		return s.Page.HTML()
	}

	el, err := s.resolveSelector(selector)
	if err != nil {
		return "", err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return el.HTML()
}

// GetOuterHTML gets the outer HTML of an element
func (s *Session) GetOuterHTML(selector string) (string, error) {
	el, err := s.resolveSelector(selector)
	if err != nil {
		return "", err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	res, err := el.Eval("el => el.outerHTML")
	if err != nil {
		return "", err
	}
	return res.Value.String(), nil
}

// GetScrollHeight gets the total scrollable height of the page
func (s *Session) GetScrollHeight() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// go-rod wraps Eval in a function internally, so we can't use Math.max() directly
	// as it tries to call .apply() on the result. Use a variable assignment instead.
	res, err := s.Page.Eval(`
		var h = document.body.scrollHeight;
		if (document.documentElement.scrollHeight > h) h = document.documentElement.scrollHeight;
		if (document.body.offsetHeight > h) h = document.body.offsetHeight;
		h
	`)
	if err != nil {
		return 0, fmt.Errorf("height eval failed: %w", err)
	}
	return int(res.Value.Num()), nil
}

// ExecuteWithRecovery executes an action with automatic retry and rollback on failure
// This implements Sprint 19: Delta Rollback & Autonomous Recovery
func (s *Session) ExecuteWithRecovery(actionName string, actionFunc func() error, confirm bool) (*ActionResult, error) {
	// If no recovery manager, just execute directly
	if s.RecoveryMgr == nil {
		err := actionFunc()
		return &ActionResult{
			Success:     err == nil,
			Error:       err,
			Timestamp:   time.Now(),
			Description: actionName,
		}, err
	}

	recoveryMgr := s.RecoveryMgr
	maxRetries := recoveryMgr.config.MaxRetriesPerAction

	// Create checkpoint before action if it's irreversible and rollback is enabled
	if confirm && recoveryMgr.config.EnableAutoRollback {
		if checkpoint, err := recoveryMgr.CreatePreActionCheckpoint(s, actionName); err == nil {
			// Checkpoint created successfully
			_ = checkpoint
		} else {
			// Log but continue - checkpoint failure shouldn't block action
			fmt.Printf("Warning: Failed to create pre-action checkpoint: %v\n", err)
		}
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Execute the action
		err := actionFunc()

		// Record the result
		failureType := recoveryMgr.DetectFailureType(err)
		result := &ActionResult{
			Success:      err == nil,
			Error:        err,
			FailureType:  failureType,
			RetriesUsed:  attempt,
			Timestamp:    time.Now(),
			Description:  actionName,
		}
		recoveryMgr.RecordAction(s.ID, result)

		if err == nil {
			return result, nil
		}

		// Action failed - check if we should retry
		if !recoveryMgr.IsRetryable(failureType) {
			// Non-retryable failure - return immediately with suggestion
			return result, fmt.Errorf("%s: %w (Suggestion: %s)",
				failureType, err,
				recoveryMgr.SuggestAlternativePath(actionName, failureType))
		}

		// Check if we've exceeded max retries
		if attempt >= maxRetries {
			// Try rollback if enabled
			if shouldRollback, cp := recoveryMgr.ShouldRollback(s.ID); shouldRollback && cp != nil {
				err = recoveryMgr.RollbackToCheckpoint(s, cp)
				if err != nil {
					return result, fmt.Errorf("max retries exceeded and rollback failed: %w", err)
				}
				return result, fmt.Errorf("max retries exceeded, rolled back to checkpoint: %w", err)
			}
			return result, fmt.Errorf("max retries (%d) exceeded: %w (Suggestion: %s)",
				maxRetries, err,
				recoveryMgr.SuggestAlternativePath(actionName, failureType))
		}

		// Calculate backoff and wait
		backoff := recoveryMgr.CalculateBackoff(attempt)
		fmt.Printf("Retry %d/%d after %v (failure: %s)\n", attempt+1, maxRetries, backoff, failureType)
		time.Sleep(backoff)

		lastErr = err
	}

	return &ActionResult{
		Success:     false,
		Error:       lastErr,
		RetriesUsed: maxRetries,
		Timestamp:   time.Now(),
		Description: actionName,
	}, lastErr
}


