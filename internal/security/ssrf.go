package security

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// SSRFEventType represents the type of SSRF event
type SSRFEventType string

const (
	// EventAllowed allowed request
	EventAllowed SSRFEventType = "allowed"
	// EventBlocked blocked request
	EventBlocked SSRFEventType = "blocked"
	// EventWarning suspicious request
	EventWarning SSRFEventType = "warning"
)

// SSRFEvent represents an SSRF guard event
type SSRFEvent struct {
	Type        SSRFEventType `json:"type"`
	Timestamp   time.Time     `json:"timestamp"`
	URL         string        `json:"url"`
	Domain      string        `json:"domain,omitempty"`
	IP          string        `json:"ip,omitempty"`
	Reason      string        `json:"reason"`
	SessionID   string        `json:"session_id,omitempty"`
}

// SSRFEventHandler is a callback for SSRF events
type SSRFEventHandler func(event *SSRFEvent)

// SSRFGuard provides SSRF protection
type SSRFGuard struct {
	allowPrivateNetwork bool
	domainAllowlist    []string
	domainDenylist     []string
	schemeAllowlist    []string
	eventHandler       SSRFEventHandler
}

// NewSSRFGuard creates a new SSRF guard
func NewSSRFGuard(allowPrivate bool, allowlist, denylist, schemes []string) *SSRFGuard {
	return &SSRFGuard{
		allowPrivateNetwork: allowPrivate,
		domainAllowlist:    allowlist,
		domainDenylist:     denylist,
		schemeAllowlist:    schemes,
	}
}

// SetEventHandler sets the event handler for SSRF events
func (g *SSRFGuard) SetEventHandler(handler SSRFEventHandler) {
	g.eventHandler = handler
}

// ValidateURL validates a URL for SSRF attacks
func (g *SSRFGuard) ValidateURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Check scheme
	if !g.isSchemeAllowed(parsed.Scheme) {
		event := &SSRFEvent{
			Type:      EventBlocked,
			Timestamp: time.Now(),
			URL:       rawURL,
			Reason:    fmt.Sprintf("scheme '%s' not allowed", parsed.Scheme),
		}
		g.emitEvent(event)
		return fmt.Errorf("scheme '%s' not allowed", parsed.Scheme)
	}

	// Block dangerous schemes
	if parsed.Scheme == "javascript" || parsed.Scheme == "data" || parsed.Scheme == "file" {
		event := &SSRFEvent{
			Type:      EventBlocked,
			Timestamp: time.Now(),
			URL:       rawURL,
			Reason:    fmt.Sprintf("scheme '%s' is not allowed", parsed.Scheme),
		}
		g.emitEvent(event)
		return fmt.Errorf("scheme '%s' is not allowed", parsed.Scheme)
	}

	// Get host
	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("empty host")
	}

	// Check for IP addresses
	ip := net.ParseIP(host)
	if ip != nil {
		err := g.validateIP(ip)
		if err != nil {
			event := &SSRFEvent{
				Type:      EventBlocked,
				Timestamp: time.Now(),
				URL:       rawURL,
				IP:        ip.String(),
				Reason:    err.Error(),
			}
			g.emitEvent(event)
			return err
		}
	}

	// Resolve DNS to check for DNS rebinding
	return g.validateDomain(host, rawURL)
}

func (g *SSRFGuard) isSchemeAllowed(scheme string) bool {
	scheme = strings.ToLower(scheme)
	for _, s := range g.schemeAllowlist {
		if strings.ToLower(s) == scheme {
			return true
		}
	}
	return false
}

func (g *SSRFGuard) validateIP(ip net.IP) error {
	// Block private IPs unless explicitly allowed
	if !g.allowPrivateNetwork {
		privateRanges := []string{
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
			"127.0.0.0/8",
			"169.254.0.0/16", // Link-local
			"0.0.0.0/8",
			"100.64.0.0/10", // Carrier-grade NAT
			"192.0.0.0/24",
			"192.0.2.0/24",  // Documentation
			"198.51.100.0/24", // Documentation
			"203.0.113.0/24", // Documentation
			"fc00::/7", // IPv6 private
			"fe80::/10", // IPv6 link-local
		}

		for _, cidr := range privateRanges {
			_, network, err := net.ParseCIDR(cidr)
			if err != nil {
				continue
			}
			if network.Contains(ip) {
				event := &SSRFEvent{
					Type:      EventBlocked,
					Timestamp: time.Now(),
					IP:        ip.String(),
					Reason:    fmt.Sprintf("private IP range %s not allowed", cidr),
				}
				g.emitEvent(event)
				return fmt.Errorf("private IP range %s not allowed", cidr)
			}
		}

		// Block loopback
		if ip.IsLoopback() {
			event := &SSRFEvent{
				Type:      EventBlocked,
				Timestamp: time.Now(),
				IP:        ip.String(),
				Reason:    "loopback IP not allowed",
			}
			g.emitEvent(event)
			return fmt.Errorf("loopback IP not allowed")
		}

		// Block unspecified
		if ip.IsUnspecified() {
			event := &SSRFEvent{
				Type:      EventBlocked,
				Timestamp: time.Now(),
				IP:        ip.String(),
				Reason:    "unspecified IP not allowed",
			}
			g.emitEvent(event)
			return fmt.Errorf("unspecified IP not allowed")
		}
	}

	return nil
}

func (g *SSRFGuard) validateDomain(domain, originalURL string) error {
	domain = strings.ToLower(domain)

	// Check denylist first
	for _, d := range g.domainDenylist {
		d = strings.ToLower(d)
		if d == domain || strings.HasSuffix(domain, "."+d) {
			event := &SSRFEvent{
				Type:      EventBlocked,
				Timestamp: time.Now(),
				URL:       originalURL,
				Domain:    domain,
				Reason:    fmt.Sprintf("domain '%s' is blocked", domain),
			}
			g.emitEvent(event)
			return fmt.Errorf("domain '%s' is blocked", domain)
		}
	}

	// Check allowlist if not empty
	if len(g.domainAllowlist) > 0 {
		allowed := false
		for _, d := range g.domainAllowlist {
			d = strings.ToLower(d)
			if d == domain || strings.HasSuffix(domain, "."+d) {
				allowed = true
				break
			}
		}
		if !allowed {
			event := &SSRFEvent{
				Type:      EventBlocked,
				Timestamp: time.Now(),
				URL:       originalURL,
				Domain:    domain,
				Reason:    fmt.Sprintf("domain '%s' not in allowlist", domain),
			}
			g.emitEvent(event)
			return fmt.Errorf("domain '%s' not in allowlist", domain)
		}
	}

	// Try to resolve and check for DNS rebinding
	ips, err := net.LookupIP(domain)
	if err != nil {
		// DNS resolution failed - could be DNS rebinding attack
		event := &SSRFEvent{
			Type:      EventWarning,
			Timestamp: time.Now(),
			URL:       originalURL,
			Domain:    domain,
			Reason:    fmt.Sprintf("DNS resolution failed for '%s'", domain),
		}
		g.emitEvent(event)
		return fmt.Errorf("DNS resolution failed for '%s'", domain)
	}

	// Check if any resolved IP is private
	if !g.allowPrivateNetwork {
		for _, ip := range ips {
			if err := g.validateIP(ip); err != nil {
				event := &SSRFEvent{
					Type:      EventBlocked,
					Timestamp: time.Now(),
					URL:       originalURL,
					Domain:    domain,
					IP:        ip.String(),
					Reason:    fmt.Sprintf("domain '%s' resolves to blocked IP: %v", domain, err),
				}
				g.emitEvent(event)
				return fmt.Errorf("domain '%s' resolves to blocked IP: %w", domain, err)
			}
		}
	}

	return nil
}

// emitEvent emits an SSRF event to the handler if configured
func (g *SSRFGuard) emitEvent(event *SSRFEvent) {
	if g.eventHandler != nil {
		g.eventHandler(event)
	}
}
