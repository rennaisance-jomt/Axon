package browser

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Element represents a page element
type Element struct {
	Ref         string   `json:"ref"`
	Type        string   `json:"type"`
	Label       string   `json:"label"`
	Placeholder string   `json:"placeholder,omitempty"`
	Intent      string   `json:"intent,omitempty"`
	Role        string   `json:"role,omitempty"`
	Selectors   []string `json:"selectors,omitempty"`
	Reversible  string   `json:"reversible"`
	Visible     bool     `json:"visible"`
	Enabled     bool     `json:"enabled"`
}

// Snapshot represents a page snapshot
type Snapshot struct {
	SessionID  string    `json:"session_id"`
	URL        string    `json:"url"`
	Title      string    `json:"title"`
	State      string    `json:"state"`
	Depth      string    `json:"depth"`
	Content    string    `json:"content"`
	Elements   []Element `json:"elements,omitempty"`
	Warnings   []Warning `json:"warnings,omitempty"`
	Timestamp  string    `json:"timestamp"`
	TokenCount int       `json:"token_count"`
}

// Warning represents a warning
type Warning struct {
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// SnapshotExtractor extracts page snapshots
type SnapshotExtractor struct {
	refCounter int
}

// NewSnapshotExtractor creates a new extractor
func NewSnapshotExtractor() *SnapshotExtractor {
	return &SnapshotExtractor{
		refCounter: 0,
	}
}

// Extract extracts a snapshot from the page
func (se *SnapshotExtractor) Extract(page *rod.Page, depth string) (*Snapshot, error) {
	se.refCounter = 0

	// Get page info
	url, _ := page.URL()
	title, _ := page.Title()

	// Extract elements using JavaScript
	elements, err := se.extractElements(page)
	if err != nil {
		return nil, fmt.Errorf("failed to extract elements: %w", err)
	}

	// Build snapshot
	snapshot := &Snapshot{
		URL:      url,
		Title:    title,
		State:    "ready",
		Depth:    depth,
		Elements: elements,
		Timestamp: "2026-02-27T00:00:00Z", // TODO: Add timestamp
	}

	// Generate content based on depth
	snapshot.Content = se.generateContent(snapshot, depth)

	// Estimate token count (rough approximation)
	snapshot.TokenCount = len(snapshot.Content) / 4

	return snapshot, nil
}

func (se *SnapshotExtractor) extractElements(page *rod.Page) ([]Element, error) {
	// Use JavaScript to extract accessibility tree
	script := `
		() => {
			const elements = [];
			const walker = document.createTreeWalker(
				document.body,
				1,
				null,
				false
			);

			let node;
			let ref = 1;
			while (node = walker.nextNode()) {
				const role = node.getAttribute('role');
				const tagName = node.tagName.toLowerCase();
				const type = node.getAttribute('type') || '';

				// Only include interactive elements
				if (role || ['input', 'button', 'a', 'select', 'textarea'].includes(tagName)) {
					const label = node.getAttribute('aria-label') ||
								  node.textContent?.substring(0, 50).trim() ||
								  node.getAttribute('name') || '';
					const placeholder = node.getAttribute('placeholder') || '';
					const disabled = node.hasAttribute('disabled');

					let elementType = tagName;
					if (tagName === 'input') {
						if (type === 'checkbox' || type === 'radio') elementType = type;
						else if (type === 'submit' || type === 'button') elementType = 'button';
						else elementType = 'textbox';
					}

					if (label || placeholder || role) {
						elements.push({
							ref: (tagName[0] || 'e') + ref++,
							type: elementType,
							label: label.substring(0, 100),
							placeholder: placeholder,
							role: role || tagName,
							visible: node.offsetParent !== null,
							enabled: !disabled
						});
					}
				}
			}
			return elements;
		}
	`

	var result []Element
	err := page.Eval(script, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (se *SnapshotExtractor) generateContent(snapshot *Snapshot, depth string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("PAGE: %s | Title: %s | State: %s\n\n", 
		strings.TrimPrefix(snapshot.URL, "https://"), snapshot.Title, snapshot.State))

	// Group elements by type
	var actions, inputs, links, navs []Element
	for _, el := range snapshot.Elements {
		switch el.Type {
		case "button", "submit":
			actions = append(actions, el)
		case "textbox", "textarea", "password", "email":
			inputs = append(inputs, el)
		case "a", "link":
			links = append(links, el)
		case "nav":
			navs = append(navs, el)
		}
	}

	// Output based on depth
	if depth == "compact" || depth == "standard" || depth == "full" {
		if len(navs) > 0 {
			sb.WriteString("NAV:\n  ")
			for i, el := range navs {
				if i > 0 sb.WriteString("  ")
				sb.WriteString(fmt.Sprintf("[%s] %s", el.Ref, el.Label))
			}
			sb.WriteString("\n\n")
		}

		if len(inputs) > 0 {
			sb.WriteString("INPUTS:\n")
			for _, el := range inputs {
				sb.WriteString(fmt.Sprintf("  [%s] %s", el.Ref, el.Label))
				if el.Placeholder != "" {
					sb.WriteString(fmt.Sprintf(" (%s)", el.Placeholder))
				}
				sb.WriteString(fmt.Sprintf(" — %s\n", el.Type))
			}
			sb.WriteString("\n")
		}

		if len(actions) > 0 {
			sb.WriteString("ACTIONS:\n")
			for _, el := range actions {
				sb.WriteString(fmt.Sprintf("  [%s] %s (%s)", el.Ref, el.Label, el.Type))
				// Mark irreversible actions
				if strings.Contains(strings.ToLower(el.Label), "delete") ||
				   strings.Contains(strings.ToLower(el.Label), "post") ||
				   strings.Contains(strings.ToLower(el.Label), "submit") {
					sb.WriteString(" [IRREVERSIBLE]")
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}

		if len(links) > 0 && depth == "full" {
			sb.WriteString("LINKS:\n")
			for _, el := range links {
				sb.WriteString(fmt.Sprintf("  [%s] %s\n", el.Ref, el.Label))
			}
		}
	}

	return sb.String()
}

// ClassifyIntent classifies element intent based on label and role
func (se *SnapshotExtractor) ClassifyIntent(label, role string) string {
	label = strings.ToLower(label)
	role = strings.ToLower(role)

	// Auth intents
	if strings.Contains(label, "login") || strings.Contains(label, "sign in") || strings.Contains(label, "signin") {
		return "auth.login"
	}
	if strings.Contains(label, "logout") || strings.Contains(label, "sign out") {
		return "auth.logout"
	}
	if strings.Contains(label, "register") || strings.Contains(label, "sign up") {
		return "auth.register"
	}

	// Search intents
	if strings.Contains(label, "search") || role == "searchbox" {
		return "search.query"
	}

	// Social intents
	if strings.Contains(label, "post") || strings.Contains(label, "tweet") || strings.Contains(label, "publish") {
		return "social.publish"
	}
	if strings.Contains(label, "like") || strings.Contains(label, "heart") {
		return "social.like"
	}
	if strings.Contains(label, "share") {
		return "social.share"
	}

	// Form intents
	if strings.Contains(label, "email") || strings.Contains(label, "e-mail") {
		return "form.email"
	}
	if strings.Contains(label, "password") {
		return "form.password"
	}
	if strings.Contains(label, "username") || strings.Contains(label, "user") {
		return "form.username"
	}

	// Content intents
	if strings.Contains(label, "delete") {
		return "content.delete"
	}
	if strings.Contains(label, "edit") || strings.Contains(label, "modify") {
		return "content.edit"
	}
	if strings.Contains(label, "save") || strings.Contains(label, "submit") {
		return "content.save"
	}

	// Navigation
	if strings.Contains(label, "home") {
		return "nav.primary"
	}
	if strings.Contains(label, "menu") {
		return "nav.menu"
	}

	return "unknown"
}

// GetElementByRef finds an element by its reference
func (se *SnapshotExtractor) GetElementByRef(snapshot *Snapshot, ref string) *Element {
	for i := range snapshot.Elements {
		if snapshot.Elements[i].Ref == ref {
			return &snapshot.Elements[i]
		}
	}
	return nil
}

// ToJSON converts snapshot to JSON
func (se *SnapshotExtractor) ToJSON(snapshot *Snapshot) (string, error) {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
