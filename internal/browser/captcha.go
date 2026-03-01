package browser

import (
	"regexp"
	"strings"
)

// CaptchaType represents different types of CAPTCHA
type CaptchaType string

const (
	CaptchaTypeNone       CaptchaType = "none"
	CaptchaTypeReCAPTCHA  CaptchaType = "recaptcha"
	CaptchaTypeHCaptcha   CaptchaType = "hcaptcha"
	CaptchaTypeImage      CaptchaType = "image"
	CaptchaTypeText       CaptchaType = "text"
	CaptchaTypeAudio      CaptchaType = "audio"
	CaptchaTypeCloudflare CaptchaType = "cloudflare"
	CaptchaTypeUnknown    CaptchaType = "unknown"
)

// CaptchaInfo represents detected CAPTCHA information
type CaptchaInfo struct {
	Detected bool        `json:"detected"`
	Type     CaptchaType `json:"type"`
	Provider string      `json:"provider,omitempty"`
	SiteKey  string      `json:"site_key,omitempty"`
	Action   string      `json:"action,omitempty"`
	Message  string      `json:"message,omitempty"`
}

// CaptchaDetector detects CAPTCHA on pages
type CaptchaDetector struct {
	patterns map[CaptchaType][]*regexp.Regexp
}

// NewCaptchaDetector creates a new CAPTCHA detector
func NewCaptchaDetector() *CaptchaDetector {
	return &CaptchaDetector{
		patterns: map[CaptchaType][]*regexp.Regexp{
			CaptchaTypeReCAPTCHA: {
				regexp.MustCompile(`google\.com/recaptcha`),
				regexp.MustCompile(`g-recaptcha`),
				regexp.MustCompile(`recaptcha.*challenge`),
				regexp.MustCompile(`data-sitekey="[^"]*recaptcha`),
			},
			CaptchaTypeHCaptcha: {
				regexp.MustCompile(`hcaptcha\.com`),
				regexp.MustCompile(`h-captcha`),
				regexp.MustCompile(`data-hcaptcha-sitekey`),
			},
			CaptchaTypeCloudflare: {
				regexp.MustCompile(`cloudflare.*challenge`),
				regexp.MustCompile(`cf-browser-verification`),
				regexp.MustCompile(`turnstile`),
				regexp.MustCompile(`cf-challenge`),
			},
			CaptchaTypeImage: {
				regexp.MustCompile(`captcha[^>]*src="[^"]*\.(png|jpg|jpeg|gif)`),
				regexp.MustCompile(`captcha.*image`),
				regexp.MustCompile(`img[^>]*captcha`),
			},
			CaptchaTypeText: {
				regexp.MustCompile(`enter.*text.*shown`),
				regexp.MustCompile(`type.*characters`),
				regexp.MustCompile(`captcha.*input`),
			},
			CaptchaTypeAudio: {
				regexp.MustCompile(`audio.*captcha`),
				regexp.MustCompile(`captcha.*audio`),
				regexp.MustCompile(`play.*sound`),
			},
		},
	}
}

// Detect analyzes HTML content for CAPTCHA presence
func (d *CaptchaDetector) Detect(htmlContent string) CaptchaInfo {
	htmlLower := strings.ToLower(htmlContent)
	
	// Check each pattern type
	for captchaType, patterns := range d.patterns {
		for _, pattern := range patterns {
			if pattern.MatchString(htmlContent) || pattern.MatchString(htmlLower) {
				return CaptchaInfo{
					Detected: true,
					Type:     captchaType,
					Provider: d.getProvider(captchaType),
					Message:  d.getMessage(captchaType),
				}
			}
		}
	}
	
	// Check for generic CAPTCHA indicators
	if d.hasGenericCaptchaIndicators(htmlLower) {
		return CaptchaInfo{
			Detected: true,
			Type:     CaptchaTypeUnknown,
			Provider: "unknown",
			Message:  "CAPTCHA-like challenge detected but type could not be determined",
		}
	}
	
	return CaptchaInfo{
		Detected: false,
		Type:     CaptchaTypeNone,
	}
}

// DetectInSnapshot checks a snapshot for CAPTCHA
func (d *CaptchaDetector) DetectInSnapshot(snapshot *Snapshot) CaptchaInfo {
	// Check the HTML content
	content := snapshot.Content
	result := d.Detect(content)
	
	// Also check element roles and types for CAPTCHA indicators
	if !result.Detected {
		for _, el := range snapshot.Elements {
			if d.isCaptchaElement(el) {
				result = CaptchaInfo{
					Detected: true,
					Type:     CaptchaTypeUnknown,
					Provider: "unknown",
					Message:  "CAPTCHA element detected in page structure",
				}
				break
			}
		}
	}
	
	return result
}

func (d *CaptchaDetector) isCaptchaElement(el Element) bool {
	captchaIndicators := []string{"captcha", "recaptcha", "hcaptcha", "turnstile", "challenge"}
	
	// Check label
	for _, indicator := range captchaIndicators {
		if containsIgnoreCase(el.Label, indicator) {
			return true
		}
	}
	
	// Check role
	for _, indicator := range captchaIndicators {
		if containsIgnoreCase(el.Role, indicator) {
			return true
		}
	}
	
	// Check intent
	for _, indicator := range captchaIndicators {
		if containsIgnoreCase(el.Intent, indicator) {
			return true
		}
	}
	
	return false
}

func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func (d *CaptchaDetector) hasGenericCaptchaIndicators(content string) bool {
	indicators := []string{
		"captcha",
		"i'm not a robot",
		"prove you're human",
		"security check",
		"verification",
		"challenge",
	}
	
	for _, indicator := range indicators {
		if strings.Contains(content, indicator) {
			// Additional check to reduce false positives
			if d.isLikelyCaptchaContext(content, indicator) {
				return true
			}
		}
	}
	
	return false
}

func (d *CaptchaDetector) isLikelyCaptchaContext(content, indicator string) bool {
	// Context clues that suggest real CAPTCHA vs just mentioning the word
	contextClues := []string{
		"input",
		"enter",
		"type",
		"click",
		"select",
		"verify",
		"submit",
		"reload",
		"refresh",
	}
	
	// Find position of indicator
	idx := strings.Index(content, indicator)
	if idx == -1 {
		return false
	}
	
	// Get surrounding context (100 chars before and after)
	start := idx - 100
	if start < 0 {
		start = 0
	}
	end := idx + len(indicator) + 100
	if end > len(content) {
		end = len(content)
	}
	
	context := content[start:end]
	
	// Check for context clues
	for _, clue := range contextClues {
		if strings.Contains(context, clue) {
			return true
		}
	}
	
	return false
}

func (d *CaptchaDetector) getProvider(captchaType CaptchaType) string {
	switch captchaType {
	case CaptchaTypeReCAPTCHA:
		return "Google"
	case CaptchaTypeHCaptcha:
		return "hCaptcha"
	case CaptchaTypeCloudflare:
		return "Cloudflare"
	default:
		return "unknown"
	}
}

func (d *CaptchaDetector) getMessage(captchaType CaptchaType) string {
	switch captchaType {
	case CaptchaTypeReCAPTCHA:
		return "Google reCAPTCHA challenge detected"
	case CaptchaTypeHCaptcha:
		return "hCaptcha challenge detected"
	case CaptchaTypeCloudflare:
		return "Cloudflare Turnstile/Challenge detected"
	case CaptchaTypeImage:
		return "Image-based CAPTCHA detected"
	case CaptchaTypeText:
		return "Text-based CAPTCHA detected"
	case CaptchaTypeAudio:
		return "Audio CAPTCHA detected"
	default:
		return "CAPTCHA challenge detected"
	}
}

// GetSupportedTypes returns a list of supported CAPTCHA types
func (d *CaptchaDetector) GetSupportedTypes() []CaptchaType {
	return []CaptchaType{
		CaptchaTypeReCAPTCHA,
		CaptchaTypeHCaptcha,
		CaptchaTypeCloudflare,
		CaptchaTypeImage,
		CaptchaTypeText,
		CaptchaTypeAudio,
	}
}
