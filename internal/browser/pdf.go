package browser

import (
	"fmt"
	"os"
	"strings"
)

// PDFOptions represents PDF export options
type PDFOptions struct {
	Scale         float64 `json:"scale,omitempty"`
	PrintBackground bool   `json:"print_background,omitempty"`
	Landscape     bool   `json:"landscape,omitempty"`
	Format        string  `json:"format,omitempty"` // A4, Letter, etc.
	Margins       Margin  `json:"margins,omitempty"`
	PageRanges    string  `json:"page_ranges,omitempty"` // "1-5, 8, 11-13"
	HeaderHTML    string  `json:"header_html,omitempty"`
	FooterHTML    string  `json:"footer_html,omitempty"`
}

// Margin represents page margins
type Margin struct {
	Top    string `json:"top,omitempty"`
	Bottom string `json:"bottom,omitempty"`
	Left   string `json:"left,omitempty"`
	Right  string `json:"right,omitempty"`
}

// ExportPDF exports the page as PDF
func (s *Session) ExportPDF(path string, opts *PDFOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if opts == nil {
		opts = &PDFOptions{}
	}

	// Validate path
	if !strings.HasSuffix(path, ".pdf") {
		path = path + ".pdf"
	}

	// Create file
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create PDF file: %w", err)
	}
	defer file.Close()

	// Get PDF with options
	data, err := s.Page.PrintToPDF()
	if err != nil {
		return fmt.Errorf("failed to generate PDF: %w", err)
	}

	// Write to file
	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write PDF: %w", err)
	}

	return nil
}

// StateDetector detects page state
type StateDetector struct{}

// NewStateDetector creates a new state detector
func NewStateDetector() *StateDetector {
	return &StateDetector{}
}

// DetectAuthState detects authentication state
func (sd *StateDetector) DetectAuthState(page *rod.Page) string {
	// Check for login forms
	script := `
		() => {
			// Check for common login form elements
			const loginIndicators = [
				'input[type="password"]',
				'input[name*="password"]',
				'input[name*="email"]',
				'form[action*="login"]',
				'form[action*="signin"]',
				'[data-testid="login"]',
				'[data-testid="signin"]',
				'button:contains("Sign In")',
				'button:contains("Login")',
				'button:contains("Log In")',
			];
			
			for (const selector of loginIndicators) {
				if (document.querySelector(selector)) {
					return 'logged_out';
				}
			}
			
			// Check for logged-in indicators
			const loggedInIndicators = [
				'[data-testid="userAvatar"]',
				'[data-testid="profile"]',
				'.user-menu',
				'.account-menu',
				'a[href*="/settings"]',
				'a[href*="/account"]',
				'a[href*="/logout"]',
				'a[href*="/signout"]',
			];
			
			for (const selector of loggedInIndicators) {
				if (document.querySelector(selector)) {
					return 'logged_in';
				}
			}
			
			return 'unknown';
		}
	`

	var result string
	err := page.Eval(script, &result)
	if err != nil {
		return "unknown"
	}

	return result
}

// DetectPageState detects page state (loading, ready, error, etc.)
func (sd *StateDetector) DetectPageState(page *rod.Page) string {
	script := `
		() => {
			// Check for loading state
			if (document.readyState === 'loading') {
				return 'loading';
			}
			
			// Check for error pages
			const body = document.body;
			if (body) {
				const text = body.innerText.toLowerCase();
				if (text.includes('error') && text.includes('404')) {
					return 'error';
				}
				if (text.includes('error') && text.includes('500')) {
					return 'error';
				}
				if (text.includes('too many requests') || text.includes('rate limit')) {
					return 'rate_limited';
				}
				if (text.includes('captcha') || text.includes('verify you are human')) {
					return 'captcha';
				}
			}
			
			return 'ready';
		}
	`

	var result string
	err := page.Eval(script, &result)
	if err != nil {
		return "unknown"
	}

	return result
}

// DetectRateLimit detects if page is rate limited
func (sd *StateDetector) DetectRateLimit(page *rod.Page) bool {
	script := `
		() => {
			const body = document.body.innerText.toLowerCase();
			return body.includes('too many requests') || 
			       body.includes('rate limit') ||
			       body.includes('429') ||
			       body.includes('please wait');
		}
	`

	var result bool
	err := page.Eval(script, &result)
	if err != nil {
		return false
	}

	return result
}

// DetectCaptcha detects if page has CAPTCHA
func (sd *StateDetector) DetectCaptcha(page *rod.Page) (bool, string) {
	script := `
		() => {
			// Check for common CAPTCHA providers
			const captchaIndicators = [
				{ selector: '.g-recaptcha', type: 'recaptcha_v2' },
				{ selector: '[data-sitekey]', type: 'recaptcha_v2' },
				{ selector: '.cf-challenge', type: 'cloudflare' },
				{ selector: '#cf-challenge', type: 'cloudflare' },
				{ selector: '.h-captcha', type: 'hcaptcha' },
				{ selector: '[data-hcaptcha]', type: 'hcaptcha' },
				{ selector: '.funcaptcha', type: 'funcaptcha' },
				{ selector: '[data-token]', type: 'custom' },
			];
			
			for (const indicator of captchaIndicators) {
				if (document.querySelector(indicator.selector)) {
					return { detected: true, type: indicator.type };
				}
			}
			
			return { detected: false, type: '' };
		}
	`

	var result struct {
		Detected bool   `json:"detected"`
		Type     string `json:"type"`
	}
	err := page.Eval(script, &result)
	if err != nil {
		return false, ""
	}

	return result.Detected, result.Type
}
