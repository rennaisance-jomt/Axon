package browser

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// Element represents a page element
type Element struct {
	Ref             string              `json:"ref"`
	RelatedRef      string              `json:"related_ref,omitempty"`
	Type            string              `json:"type"`
	Label           string              `json:"label"`
	Placeholder     string              `json:"placeholder,omitempty"`
	Intent          string              `json:"intent,omitempty"`
	Role            string              `json:"role,omitempty"`
	Selectors       []string            `json:"selectors,omitempty"`
	Reversible      string              `json:"reversible"`
	Visible         bool                `json:"visible"`
	Enabled         bool                `json:"enabled"`
	BackendNodeID   proto.DOMBackendNodeID `json:"-"` // Internal: BackendDOMNodeID for element actions
}

// Snapshot represents a page snapshot
type Snapshot struct {
	SessionID  string    `json:"session_id"`
	URL        string    `json:"url"`
	Title      string    `json:"title"`
	State      string    `json:"state"`
	AuthState  string    `json:"auth_state,omitempty"`
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
func (se *SnapshotExtractor) Extract(page *rod.Page, depth string, selector string) (*Snapshot, error) {
	se.refCounter = 0

	// Get page info
	info := page.MustInfo()
	url := info.URL
	
	var title string
	if res, err := page.Eval("document.title"); err == nil {
		title = res.Value.String()
	}

	// Extract elements using JavaScript
	elements, err := se.extractElements(page, selector)
	if err != nil {
		return nil, fmt.Errorf("failed to extract elements: %w", err)
	}

	// Detect page state
	detector := NewStateDetector()
	pageState := detector.DetectPageState(page)
	authState := detector.DetectAuthState(page)

	// Classify elements
	for i := range elements {
		elements[i].Intent = se.ClassifyIntent(elements[i].Label, elements[i].Role)
		
		// Basic reversibility inference
		if strings.HasPrefix(elements[i].Intent, "content.delete") || 
		   strings.HasPrefix(elements[i].Intent, "social.publish") {
			elements[i].Reversible = "write_irreversible"
		} else if strings.HasPrefix(elements[i].Intent, "content.") || 
		           strings.HasPrefix(elements[i].Intent, "auth.") {
			elements[i].Reversible = "write_reversible"
		} else {
			elements[i].Reversible = "read"
		}
	}

	// Collapse related elements into High-Compression Intent Graphs
	elements = se.CollapseIntentGraph(elements)

	// Build snapshot
	snapshot := &Snapshot{
		URL:       url,
		Title:     title,
		State:     pageState, // Use detected page state
		AuthState: authState,
		Depth:     depth,
		Elements:  elements,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Generate content based on depth
	snapshot.Content = se.generateContent(snapshot, depth)

	// Estimate token count (rough approximation)
	snapshot.TokenCount = len(snapshot.Content) / 4

	return snapshot, nil
}

func (se *SnapshotExtractor) extractElements(page *rod.Page, selector string) ([]Element, error) {
	// Enable DOM domain first to ensure BackendNodeId operations work
	_ = proto.DOMEnable{}.Call(page)

	// Fetch native C++ Accessibility Tree
	req := &proto.AccessibilityGetFullAXTree{}
	res, err := req.Call(page)
	if err != nil {
		return nil, fmt.Errorf("accessibility tree failed: %w", err)
	}

	var elements []Element
	refID := 1

	for _, node := range res.Nodes {
		if node.Ignored {
			continue
		}

		role := ""
		if node.Role != nil {
			role = node.Role.Value.String()
		}

		// Only process interactive roles
		if role == "button" || role == "link" || role == "textbox" || role == "searchbox" || role == "checkbox" || role == "radio" || role == "combobox" || role == "listbox" {
			name := ""
			if node.Name != nil {
				name = node.Name.Value.String()
			}
			
			disabled := false
			var placeholder string
			
			for _, prop := range node.Properties {
				if prop.Name == proto.AccessibilityAXPropertyNameDisabled && !prop.Value.Value.Nil() {
					disabled = prop.Value.Value.Bool()
				}
				if string(prop.Name) == "placeholder" && !prop.Value.Value.Nil() {
					placeholder = prop.Value.Value.String()
				}
			}
			
			elementType := role
			if role == "searchbox" {
				elementType = "textbox"
			}
			if role == "combobox" || role == "listbox" {
				elementType = "select"
			}
			
			char := "e"
			if len(elementType) > 0 {
				char = elementType[:1]
			}
			ref := fmt.Sprintf("%s%d", char, refID)
			refID++
			
			// Make label shorter
			if len(name) > 100 {
				name = name[:100]
			}

			elements = append(elements, Element{
				Ref:         ref,
				Type:        elementType,
				Label:       name,
				Placeholder: placeholder,
				Role:        role,
				Visible:     true, // Accessibility nodes exposed in this tree are generally visible
				Enabled:     !disabled,
				BackendNodeID: node.BackendDOMNodeID, // Store for element actions
			})
			
			// Debug: Log if BackendNodeID is available
			if node.BackendDOMNodeID > 0 {
				log.Printf("[snapshot] Element %s has BackendNodeID: %d", ref, node.BackendDOMNodeID)
			}
		}
	}
	
	return elements, nil
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
		case "textbox", "textarea", "password", "email", "searchbox", "input_group":
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
				if i > 0 {
					sb.WriteString("  ")
				}
				sb.WriteString(fmt.Sprintf("[%s] %s", el.Ref, el.Label))
			}
			sb.WriteString("\n\n")
		}

		if len(inputs) > 0 {
			sb.WriteString("INPUTS:\n")
			for _, el := range inputs {
				if el.RelatedRef != "" {
					sb.WriteString(fmt.Sprintf("  [%s|%s] %s", el.Ref, el.RelatedRef, el.Label))
				} else {
					sb.WriteString(fmt.Sprintf("  [%s] %s", el.Ref, el.Label))
				}
				
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

// CollapseIntentGraph reduces token cost by semantically grouping localized elements
func (se *SnapshotExtractor) CollapseIntentGraph(elements []Element) []Element {
	if len(elements) == 0 {
		return elements
	}

	var collapsed []Element
	
	for i := 0; i < len(elements); i++ {
		curr := elements[i]

		// Pattern: Text input immediately followed by a button
		if (curr.Type == "textbox" || curr.Type == "searchbox" || curr.Type == "email" || curr.Type == "password") && i+1 < len(elements) {
			next := elements[i+1]
			
			if next.Type == "button" || next.Type == "submit" {
				// Compress into an intent group
				curr.Type = "input_group"
				if curr.Label == "" {
					curr.Label = next.Label
				} else if next.Label != "" && next.Label != curr.Label {
					curr.Label = curr.Label + " / " + next.Label
				}
				
				curr.RelatedRef = next.Ref
				curr.Role = "group"
				
				// Re-classify the combined intent
				curr.Intent = se.ClassifyIntent(curr.Label, curr.Role)

				collapsed = append(collapsed, curr)
				i++ // skip the now-merged button
				continue
			}
		}

		collapsed = append(collapsed, curr)
	}

	return collapsed
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
