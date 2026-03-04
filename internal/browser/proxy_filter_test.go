package browser

import (
	"testing"
)

func TestProxyFilter_ShouldFilter(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectBlock bool
	}{
		// Should be BLOCKED - Ads
		{"DoubleClick ad", "https://doubleclick.net/ad/banner", true},
		{"Google syndication", "https://googlesyndication.com/ads", true},
		{"AppNexus ad", "https://adnxs.com/adserving", true},

		// Should be BLOCKED - Tracking
		{"Google Analytics", "https://www.google-analytics.com/collect", true},
		{"Facebook tracking", "https://www.facebook.com/tr", true},
		{"Hotjar tracking", "https://script.hotjar.com/tracking", true},

		// Should be BLOCKED - Analytics
		{"New Relic", "https://js-agent.newrelic.com/analytics", true},

		// Should be ALLOWED - Main content
		{"Google homepage", "https://www.google.com", false},
		{"GitHub", "https://github.com", false},
		{"Main image", "https://example.com/images/hero.jpg", false},
		{"Main JS", "https://example.com/app.js", false},
		{"Main CSS", "https://example.com/style.css", false},
		{"API call", "https://api.example.com/data", false},
	}

	// Create filter with ads and tracking enabled
	cfg := &ProxyFilterConfig{
		Enabled:        true,
		Categories:     []ProxyFilterCategory{CategoryAds, CategoryTracking, CategoryAnalytics},
		BlockByDefault: true,
	}

	filter, err := NewProxyFilter(cfg)
	if err != nil {
		t.Fatalf("Failed to create proxy filter: %v", err)
	}

	passed := 0
	failed := 0

	for _, tc := range tests {
		result, err := filter.ShouldFilter(tc.url)
		if err != nil {
			t.Errorf("ShouldFilter error for %s: %v", tc.name, err)
			continue
		}

		blocked := result.Action == ActionBlocked

		if blocked == tc.expectBlock {
			passed++
			t.Logf("✅ PASS: %s - Blocked: %v (expected: %v)", tc.name, blocked, tc.expectBlock)
		} else {
			failed++
			t.Logf("❌ FAIL: %s - Blocked: %v (expected: %v) - Reason: %s", tc.name, blocked, tc.expectBlock, result.Reason)
		}
	}

	if failed > 0 {
		t.Errorf("Proxy filter tests failed: %d failed out of %d", failed, len(tests))
	} else {
		t.Logf("✅ All %d proxy filter tests passed!", passed)
	}
}

func TestProxyFilter_Whitelist(t *testing.T) {
	cfg := &ProxyFilterConfig{
		Enabled:        true,
		Categories:     []ProxyFilterCategory{CategoryAds, CategoryTracking},
		BlockByDefault: true,
		Whitelist:     []string{"allowed-ads.example.com"},
	}

	filter, err := NewProxyFilter(cfg)
	if err != nil {
		t.Fatalf("Failed to create proxy filter: %v", err)
	}

	// This should be allowed because it's whitelisted
	result, err := filter.ShouldFilter("https://allowed-ads.example.com/banner")
	if err != nil {
		t.Errorf("ShouldFilter error: %v", err)
	}

	if result.Action != ActionAllowed {
		t.Errorf("Expected whitelisted domain to be allowed, got: %v", result.Action)
	}
}
