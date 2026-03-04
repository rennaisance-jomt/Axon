package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ProxyFilterCategory represents categories of content to filter
type ProxyFilterCategory string

const (
	// CategoryAds filters advertising
	CategoryAds ProxyFilterCategory = "ads"
	// CategoryTracking filters tracking scripts
	CategoryTracking ProxyFilterCategory = "tracking"
	// CategoryMalware filters malware domains
	CategoryMalware ProxyFilterCategory = "malware"
	// CategorySocial filters social media trackers
	CategorySocial ProxyFilterCategory = "social"
	// CategoryAnalytics filters analytics
	CategoryAnalytics ProxyFilterCategory = "analytics"
)

// IntentAction represents the inferred intent action
type IntentAction string

const (
	// ActionAllowed allowed to proceed
	ActionAllowed IntentAction = "allowed"
	// ActionBlocked blocked
	ActionBlocked IntentAction = "blocked"
	// ActionWarn warned but allowed
	ActionWarn IntentAction = "warn"
	// ActionInspect inspect more closely
	ActionInspect IntentAction = "inspect"
)

// FilterResult represents the result of filtering
type FilterResult struct {
	URL         string              `json:"url"`
	Action      IntentAction        `json:"action"`
	Category    ProxyFilterCategory `json:"category,omitempty"`
	Confidence  float64             `json:"confidence"`
	Reason      string              `json:"reason"`
	Timestamp   time.Time           `json:"timestamp"`
}

// ProxyFilterConfig holds proxy filter configuration
type ProxyFilterConfig struct {
	Enabled            bool
	Categories         []ProxyFilterCategory
	BlockByDefault     bool
	IntentConfidence   float64
	Whitelist          []string
	Blacklist          []string
	UseIntentAnalysis  bool
}

// ProxyFilter manages intent-based network filtering
type ProxyFilter struct {
	mu              sync.RWMutex
	config          *ProxyFilterConfig
	domainMatcher   *DomainMatcher
	intentAnalyzer  *IntentAnalyzer
	blockedDomains  map[string]*FilterResult
	allowedDomains  map[string]*FilterResult
	stats           *FilterStats
}

// DomainMatcher matches domains against patterns
type DomainMatcher struct {
	mu       sync.RWMutex
	patterns []*DomainPattern
}

// DomainPattern represents a domain matching pattern
type DomainPattern struct {
	Pattern   string
	Regex     *regexp.Regexp
	Category  ProxyFilterCategory
	Weight    float64
}

// IntentAnalyzer analyzes request intent
type IntentAnalyzer struct {
	mu      sync.RWMutex
	domain  string
	apiKey  string
}

// FilterStats holds filtering statistics
type FilterStats struct {
	mu                   sync.RWMutex
	TotalRequests        int64
	BlockedRequests      int64
	AllowedRequests      int64
	WarnedRequests       int64
	RequestsByCategory   map[ProxyFilterCategory]int64
	LastUpdated          time.Time
}

// NewProxyFilter creates a new proxy filter
func NewProxyFilter(cfg *ProxyFilterConfig) (*ProxyFilter, error) {
	if cfg == nil {
		cfg = &ProxyFilterConfig{
			Enabled:           true,
			Categories:        []ProxyFilterCategory{CategoryAds, CategoryTracking, CategoryMalware},
			BlockByDefault:    true,
			IntentConfidence:  0.7,
			UseIntentAnalysis: true,
		}
	}

	pf := &ProxyFilter{
		config:         cfg,
		domainMatcher:  NewDomainMatcher(),
		intentAnalyzer: &IntentAnalyzer{},
		blockedDomains: make(map[string]*FilterResult),
		allowedDomains: make(map[string]*FilterResult),
		stats: &FilterStats{
			RequestsByCategory: make(map[ProxyFilterCategory]int64),
		},
	}

	// Initialize default patterns
	pf.initDefaultPatterns()

	// Add whitelist/blacklist
	for _, d := range cfg.Whitelist {
		pf.allowedDomains[d] = &FilterResult{
			URL:        d,
			Action:     ActionAllowed,
			Confidence: 1.0,
			Reason:     "whitelisted",
		}
	}

	for _, d := range cfg.Blacklist {
		pf.blockedDomains[d] = &FilterResult{
			URL:        d,
			Action:     ActionBlocked,
			Confidence: 1.0,
			Reason:     "blacklisted",
		}
	}

	return pf, nil
}

func (pf *ProxyFilter) initDefaultPatterns() {
	patterns := []*DomainPattern{
		// Ads
		{Pattern: `.*\.doubleclick\.net`, Category: CategoryAds, Weight: 0.9},
		{Pattern: `doubleclick\.net`, Category: CategoryAds, Weight: 0.9},
		{Pattern: `.*\.googlesyndication\.com`, Category: CategoryAds, Weight: 0.9},
		{Pattern: `googlesyndication\.com`, Category: CategoryAds, Weight: 0.9},
		{Pattern: `.*\.adservice\.google\.`, Category: CategoryAds, Weight: 0.8},
		{Pattern: `.*\.adnxs\.com`, Category: CategoryAds, Weight: 0.9},
		{Pattern: `adnxs\.com`, Category: CategoryAds, Weight: 0.9},
		{Pattern: `.*\.amazon-adsystem\.com`, Category: CategoryAds, Weight: 0.8},

		// Tracking
		{Pattern: `.*\.google-analytics\.com`, Category: CategoryTracking, Weight: 0.9},
		{Pattern: `google-analytics\.com`, Category: CategoryTracking, Weight: 0.9},
		{Pattern: `facebook`, Category: CategoryTracking, Weight: 0.9},
		{Pattern: `connect\.facebook\.net`, Category: CategoryTracking, Weight: 0.8},
		{Pattern: `.*\.hotjar\.com`, Category: CategoryTracking, Weight: 0.8},
		{Pattern: `hotjar\.com`, Category: CategoryTracking, Weight: 0.8},
		{Pattern: `.*\.segment\.io`, Category: CategoryTracking, Weight: 0.8},
		{Pattern: `.*\.mixpanel\.com`, Category: CategoryTracking, Weight: 0.8},

		// Malware
		{Pattern: `.*\.malware-domain\.com`, Category: CategoryMalware, Weight: 1.0},
		{Pattern: `.*\.suspicious\.`, Category: CategoryMalware, Weight: 0.7},

		// Social trackers
		{Pattern: `.*\.addthis\.com`, Category: CategorySocial, Weight: 0.7},
		{Pattern: `.*\.sharethis\.com`, Category: CategorySocial, Weight: 0.7},

		// Analytics
		{Pattern: `.*\.newrelic\.com`, Category: CategoryAnalytics, Weight: 0.5},
		{Pattern: `newrelic\.com`, Category: CategoryAnalytics, Weight: 0.5},
		{Pattern: `.*\.datadog\.`, Category: CategoryAnalytics, Weight: 0.5},
	}

	for _, p := range patterns {
		re, err := regexp.Compile(p.Pattern)
		if err != nil {
			continue
		}
		p.Regex = re
		pf.domainMatcher.patterns = append(pf.domainMatcher.patterns, p)
	}
}

// NewDomainMatcher creates a new domain matcher
func NewDomainMatcher() *DomainMatcher {
	return &DomainMatcher{
		patterns: make([]*DomainPattern, 0),
	}
}

// ShouldFilter determines if a URL should be filtered
func (pf *ProxyFilter) ShouldFilter(requestURL string) (*FilterResult, error) {
	pf.mu.RLock()
	defer pf.mu.RUnlock()

	if !pf.config.Enabled {
		return &FilterResult{
			URL:        requestURL,
			Action:     ActionAllowed,
			Confidence: 1.0,
			Reason:     "filtering disabled",
		}, nil
	}

	// Parse URL
	parsed, err := url.Parse(requestURL)
	if err != nil {
		return &FilterResult{
			URL:        requestURL,
			Action:     ActionAllowed,
			Confidence: 0.0,
			Reason:     "invalid URL",
		}, nil
	}

	domain := parsed.Hostname()

	// Check whitelist first
	if result, ok := pf.allowedDomains[domain]; ok {
		return result, nil
	}

	// Check blacklist
	if result, ok := pf.blockedDomains[domain]; ok {
		pf.updateStats(result)
		return result, nil
	}

	// Check against patterns
	result := pf.checkDomain(domain, requestURL)

	// Update stats
	pf.updateStats(result)

	return result, nil
}

func (pf *ProxyFilter) checkDomain(domain, requestURL string) *FilterResult {
	// Match against patterns
	for _, pattern := range pf.domainMatcher.patterns {
		if pattern.Regex.MatchString(domain) {
			// Check if category is enabled
			categoryEnabled := false
			for _, cat := range pf.config.Categories {
				if cat == pattern.Category {
					categoryEnabled = true
					break
				}
			}

			if !categoryEnabled {
				continue
			}

			action := ActionBlocked
			if !pf.config.BlockByDefault {
				action = ActionWarn
			}

			return &FilterResult{
				URL:        requestURL,
				Action:     action,
				Category:   pattern.Category,
				Confidence: pattern.Weight,
				Reason:     fmt.Sprintf("matched %s pattern", pattern.Category),
				Timestamp:  time.Now(),
			}
		}
	}

	// Use intent analysis if enabled
	if pf.config.UseIntentAnalysis {
		intentResult := pf.intentAnalyzer.analyze(domain, requestURL)
		if intentResult != nil {
			return intentResult
		}
	}

	// Default allow
	return &FilterResult{
		URL:        requestURL,
		Action:     ActionAllowed,
		Confidence: 1.0,
		Reason:     "no match",
		Timestamp:  time.Now(),
	}
}

func (ia *IntentAnalyzer) analyze(domain, requestURL string) *FilterResult {
	// Simplified intent analysis - in production this would use ML
	// Check for suspicious patterns

	suspiciousPatterns := []string{
		"redirect",
		"track",
		"pixel",
		"beacon",
	}

	lowerURL := strings.ToLower(requestURL)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerURL, pattern) {
			return &FilterResult{
				URL:        requestURL,
				Action:     ActionInspect,
				Confidence: 0.6,
				Reason:     "suspicious intent pattern: " + pattern,
				Timestamp:  time.Now(),
			}
		}
	}

	return nil
}

func (pf *ProxyFilter) updateStats(result *FilterResult) {
	pf.stats.mu.Lock()
	defer pf.stats.mu.Unlock()

	pf.stats.TotalRequests++

	switch result.Action {
	case ActionBlocked:
		pf.stats.BlockedRequests++
	case ActionAllowed:
		pf.stats.AllowedRequests++
	case ActionWarn:
		pf.stats.WarnedRequests++
	}

	if result.Category != "" {
		pf.stats.RequestsByCategory[result.Category]++
	}

	pf.stats.LastUpdated = time.Now()
}

// GetStats returns filtering statistics
func (pf *ProxyFilter) GetStats() *FilterStats {
	pf.stats.mu.RLock()
	defer pf.stats.mu.RUnlock()

	stats := &FilterStats{
		TotalRequests:      pf.stats.TotalRequests,
		BlockedRequests:   pf.stats.BlockedRequests,
		AllowedRequests:   pf.stats.AllowedRequests,
		WarnedRequests:    pf.stats.WarnedRequests,
		RequestsByCategory: make(map[ProxyFilterCategory]int64),
		LastUpdated:       pf.stats.LastUpdated,
	}

	for k, v := range pf.stats.RequestsByCategory {
		stats.RequestsByCategory[k] = v
	}

	return stats
}

// AddToWhitelist adds a domain to the whitelist
func (pf *ProxyFilter) AddToWhitelist(domain string) {
	pf.mu.Lock()
	defer pf.mu.Unlock()

	pf.allowedDomains[domain] = &FilterResult{
		URL:        domain,
		Action:     ActionAllowed,
		Confidence: 1.0,
		Reason:     "user whitelisted",
	}
}

// AddToBlacklist adds a domain to the blacklist
func (pf *ProxyFilter) AddToBlacklist(domain string) {
	pf.mu.Lock()
	defer pf.mu.Unlock()

	pf.blockedDomains[domain] = &FilterResult{
		URL:        domain,
		Action:     ActionBlocked,
		Confidence: 1.0,
		Reason:     "user blacklisted",
	}
}

// InspectRequest performs deeper inspection of a request
func (pf *ProxyFilter) InspectRequest(ctx context.Context, requestData map[string]interface{}) (*FilterResult, error) {
	// In production, this would analyze request headers, body, etc.
	// For now, return a simple inspection result

	requestURL, ok := requestData["url"].(string)
	if !ok {
		return &FilterResult{
			Action:     ActionAllowed,
			Confidence: 0.0,
			Reason:     "no URL in request",
		}, nil
	}

	return pf.ShouldFilter(requestURL)
}

// MarshalJSON implements json.Marshaler for FilterStats
func (fs *FilterStats) MarshalJSON() ([]byte, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	return json.Marshal(map[string]interface{}{
		"total_requests":       fs.TotalRequests,
		"blocked_requests":     fs.BlockedRequests,
		"allowed_requests":     fs.AllowedRequests,
		"warned_requests":      fs.WarnedRequests,
		"requests_by_category": fs.RequestsByCategory,
		"last_updated":         fs.LastUpdated,
	})
}
