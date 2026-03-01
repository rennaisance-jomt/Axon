package browser

import (
	"github.com/go-rod/rod"
)

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
			const loginIndicators = [
				'input[type="password"]',
				'input[name*="password"]',
				'input[name*="email"]',
				'form[action*="login"]',
				'form[action*="signin"]',
				'[data-testid="login"]',
				'[data-testid="signin"]'
			];
			
			for (const selector of loginIndicators) {
				if (document.querySelector(selector)) {
					return 'logged_out';
				}
			}
			
			const loggedInIndicators = [
				'[data-testid="userAvatar"]',
				'[data-testid="profile"]',
				'.user-menu',
				'a[href*="/logout"]'
			];
			
			for (const selector of loggedInIndicators) {
				if (document.querySelector(selector)) {
					return 'logged_in';
				}
			}
			
			return 'unknown';
		}
	`

	res, err := page.Eval(script)
	if err != nil {
		return "unknown"
	}

	return res.Value.String()
}

// DetectPageState detects page state (loading, ready, error, etc.)
func (sd *StateDetector) DetectPageState(page *rod.Page) string {
	script := `
		() => {
			if (document.readyState === 'loading') {
				return 'loading';
			}
			
			const body = document.body;
			if (body) {
				const text = body.innerText.toLowerCase();
				if (text.includes('error') && text.includes('404')) return 'error';
				if (text.includes('too many requests') || text.includes('rate limit')) return 'rate_limited';
				if (text.includes('captcha') || text.includes('verify you are human')) return 'captcha';
			}
			
			return 'ready';
		}
	`

	res, err := page.Eval(script)
	if err != nil {
		return "unknown"
	}

	return res.Value.String()
}

// DetectRateLimit detects if page is rate limited
func (sd *StateDetector) DetectRateLimit(page *rod.Page) bool {
	script := `
		() => {
			const body = document.body.innerText.toLowerCase();
			return body.includes('too many requests') || 
			       body.includes('rate limit') ||
			       body.includes('429');
		}
	`

	res, err := page.Eval(script)
	if err != nil {
		return false
	}

	return res.Value.Bool()
}

// DetectCaptcha detects if page has CAPTCHA
func (sd *StateDetector) DetectCaptcha(page *rod.Page) (bool, string) {
	script := `
		() => {
			const captchaIndicators = [
				{ selector: '.g-recaptcha', type: 'recaptcha_v2' },
				{ selector: '.cf-challenge', type: 'cloudflare' },
				{ selector: '.h-captcha', type: 'hcaptcha' }
			];
			
			for (const indicator of captchaIndicators) {
				if (document.querySelector(indicator.selector)) {
					return { detected: true, type: indicator.type };
				}
			}
			
			return { detected: false, type: '' };
		}
	`

	res, err := page.Eval(script)
	if err != nil {
		return false, ""
	}

	detected := res.Value.Get("detected").Bool()
	captchaType := res.Value.Get("type").String()

	return detected, captchaType
}
