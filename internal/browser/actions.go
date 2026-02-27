package browser

import (
	"fmt"
	"time"

	"github.com/go-rod/rod"
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
		opts.Timeout = 30 * time.Second
	}

	switch opts.Condition {
	case "load":
		return s.Page.WaitLoad()
	case "networkidle":
		return s.Page.WaitIdle(5 * time.Second)
	case "domcontentloaded":
		return s.Page.WaitEvent("DOMContentLoaded", func() bool { return true })
	case "selector":
		if opts.Selector == "" {
			return fmt.Errorf("selector required for wait condition")
		}
		_, err := s.Page.Element(opts.Selector)
		return err
	case "text":
		if opts.Text == "" {
			return fmt.Errorf("text required for wait condition")
		}
		return s.Page.WaitForSelector(opts.Text, 0)
	}

	return fmt.Errorf("unknown wait condition: %s", opts.Condition)
}

// Hover hovers over an element
func (s *Session) Hover(selector string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	element, err := s.Page.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found: %w", err)
	}

	return element.Hover()
}

// Scroll scrolls an element or page
func (s *Session) Scroll(selector string, y int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if selector == "" {
		// Scroll page
		_, err := s.Page.Evaluate(fmt.Sprintf("window.scrollBy(0, %d)", y))
		return err
	}

	element, err := s.Page.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found: %w", err)
	}

	return element.ScrollIntoView()
}

// DoubleClick performs a double click
func (s *Session) DoubleClick(selector string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	element, err := s.Page.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found: %w", err)
	}

	return element.DoubleClick()
}

// RightClick performs a right click (context menu)
func (s *Session) RightClick(selector string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	element, err := s.Page.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found: %w", err)
	}

	return element.Click(rod.ButtonRight)
}

// SelectOption selects an option in a dropdown
func (s *Session) SelectOption(selector string, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	element, err := s.Page.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found: %w", err)
	}

	return element.Select(value, false)
}

// GetPageTitle gets the current page title
func (s *Session) GetPageTitle() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Page.Title()
}

// GetPageURL gets the current page URL
func (s *Session) GetPageURL() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.URL
}

// IsElementVisible checks if an element is visible
func (s *Session) IsElementVisible(selector string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	element, err := s.Page.Element(selector)
	if err != nil {
		return false, err
	}

	visible, err := element.IsVisible()
	return visible, err
}

// IsElementEnabled checks if an element is enabled
func (s *Session) IsElementEnabled(selector string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	element, err := s.Page.Element(selector)
	if err != nil {
		return false, err
	}

	enabled, err := element.IsDialed()
	return !enabled, err // Inverted because IsDialed returns disabled state
}

// GetElementText gets text content of an element
func (s *Session) GetElementText(selector string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	element, err := s.Page.Element(selector)
	if err != nil {
		return "", err
	}

	return element.Text()
}

// GetElementAttribute gets an attribute of an element
func (s *Session) GetElementAttribute(selector, attr string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	element, err := s.Page.Element(selector)
	if err != nil {
		return "", err
	}

	return element.Attribute(attr)
}

// Focus focuses an element
func (s *Session) Focus(selector string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	element, err := s.Page.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found: %w", err)
	}

	return element.Focus()
}

// Blur removes focus from an element
func (s *Session) Blur(selector string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.Page.Evaluate(fmt.Sprintf("document.querySelector('%s').blur()", selector))
	return err
}

// GetHTML gets the HTML of an element or page
func (s *Session) GetHTML(selector string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if selector == "" {
		return s.Page.HTML()
	}

	element, err := s.Page.Element(selector)
	if err != nil {
		return "", err
	}

	return element.HTML()
}

// GetOuterHTML gets the outer HTML of an element
func (s *Session) GetOuterHTML(selector string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	element, err := s.Page.Element(selector)
	if err != nil {
		return "", err
	}

	return element.OuterHTML()
}
