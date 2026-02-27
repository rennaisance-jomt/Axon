package security

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// SSRFGuard provides SSRF protection
type SSRFGuard struct {
	allowPrivateNetwork bool
	domainAllowlist    []string
	domainDenylist     []string
	schemeAllowlist    []string
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

// ValidateURL validates a URL for SSRF attacks
func (g *SSRFGuard) ValidateURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Check scheme
	if !g.isSchemeAllowed(parsed.Scheme) {
		return fmt.Errorf("scheme '%s' not allowed", parsed.Scheme)
	}

	// Block dangerous schemes
	if parsed.Scheme == "javascript" || parsed.Scheme == "data" || parsed.Scheme == "file" {
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
		return g.validateIP(ip)
	}

	// Resolve DNS to check for DNS rebinding
	return g.validateDomain(host)
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
				return fmt.Errorf("private IP range %s not allowed", cidr)
			}
		}

		// Block loopback
		if ip.IsLoopback() {
			return fmt.Errorf("loopback IP not allowed")
		}

		// Block unspecified
		if ip.IsUnspecified() {
			return fmt.Errorf("unspecified IP not allowed")
		}
	}

	return nil
}

func (g *SSRFGuard) validateDomain(domain string) error {
	domain = strings.ToLower(domain)

	// Check denylist first
	for _, d := range g.domainDenylist {
		d = strings.ToLower(d)
		if d == domain || strings.HasSuffix(domain, "."+d) {
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
			return fmt.Errorf("domain '%s' not in allowlist", domain)
		}
	}

	// Try to resolve and check for DNS rebinding
	ips, err := net.LookupIP(domain)
	if err != nil {
		// DNS resolution failed - could be DNS rebinding attack
		return fmt.Errorf("DNS resolution failed for '%s'", domain)
	}

	// Check if any resolved IP is private
	if !g.allowPrivateNetwork {
		for _, ip := range ips {
			if err := g.validateIP(ip); err != nil {
				return fmt.Errorf("domain '%s' resolves to blocked IP: %w", domain, err)
			}
		}
	}

	return nil
}
