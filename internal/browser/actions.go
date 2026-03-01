package browser

import (
	"fmt"
	"time"

	"github.com/go-rod/rod/lib/proto"
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
	s.mu.Lock()
	defer s.mu.Unlock()

	el, err := s.Page.Element(selector)
	if err != nil {
		return err
	}
	el.MustWaitVisible().MustWaitStable()
	return el.Hover()
}

// Scroll scrolls an element or page
func (s *Session) Scroll(selector string, y int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if selector == "" {
		_, err := s.Page.Eval(fmt.Sprintf("window.scrollBy(0, %d)", y))
		return err
	}

	el, err := s.Page.Element(selector)
	if err != nil {
		return err
	}
	el.MustWaitVisible().MustWaitStable()
	return el.ScrollIntoView()
}

// DoubleClick performs a double click
func (s *Session) DoubleClick(selector string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	el, err := s.Page.Element(selector)
	if err != nil {
		return err
	}
	el.MustWaitVisible().MustWaitStable()
	return el.Click(proto.InputMouseButtonLeft, 2)
}

// RightClick performs a right click (context menu)
func (s *Session) RightClick(selector string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	el, err := s.Page.Element(selector)
	if err != nil {
		return err
	}
	el.MustWaitVisible().MustWaitStable()
	return el.Click(proto.InputMouseButtonRight, 1)
}

// SelectOption selects an option in a dropdown
func (s *Session) SelectOption(selector string, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	el, err := s.Page.Element(selector)
	if err != nil {
		return err
	}
	el.MustWaitVisible().MustWaitStable()
	// Select by value using JS
	_, err = el.Eval(fmt.Sprintf(`el => { el.value = "%s"; el.dispatchEvent(new Event('change', { bubbles: true })); }`, value))
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
	s.mu.RLock()
	defer s.mu.RUnlock()

	el, err := s.Page.Element(selector)
	if err != nil {
		return false, err
	}

	return el.Visible()
}

// IsElementEnabled checks if an element is enabled
func (s *Session) IsElementEnabled(selector string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	el, err := s.Page.Element(selector)
	if err != nil {
		return false, err
	}

	res, err := el.Eval("el => el.disabled")
	if err != nil {
		return false, err
	}
	return !res.Value.Bool(), nil
}

// GetElementText gets text content of an element
func (s *Session) GetElementText(selector string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	el, err := s.Page.Element(selector)
	if err != nil {
		return "", err
	}

	return el.Text()
}

// GetElementAttribute gets an attribute of an element
func (s *Session) GetElementAttribute(selector, attr string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	el, err := s.Page.Element(selector)
	if err != nil {
		return "", err
	}

	val, err := el.Attribute(attr)
	if err != nil {
		return "", err
	}
	if val == nil {
		return "", nil
	}
	return *val, nil
}

// Focus focuses an element
func (s *Session) Focus(selector string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	el, err := s.Page.Element(selector)
	if err != nil {
		return err
	}
	el.MustWaitVisible().MustWaitStable()
	return el.Focus()
}

// Blur removes focus from an element
func (s *Session) Blur(selector string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.Page.Eval(fmt.Sprintf("document.querySelector('%s').blur()", selector))
	return err
}

// GetHTML gets the HTML of an element or page
func (s *Session) GetHTML(selector string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if selector == "" {
		return s.Page.HTML()
	}

	el, err := s.Page.Element(selector)
	if err != nil {
		return "", err
	}

	return el.HTML()
}

// GetOuterHTML gets the outer HTML of an element
func (s *Session) GetOuterHTML(selector string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	el, err := s.Page.Element(selector)
	if err != nil {
		return "", err
	}

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
