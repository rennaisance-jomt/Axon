package browser

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// AXAlignment combines accessibility tree with spatial data
type AXAlignment struct {
	mu         sync.RWMutex
	SpatialMap *SpatialMap
	Elements   []*AlignedElement
}

// AlignedElement combines spatial and accessibility data
type AlignedElement struct {
	Ref             string  `json:"ref"`
	BackendNodeID   int     `json:"backend_node_id"`
	Role            string  `json:"role"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Value           string  `json:"value"`
	TagName         string  `json:"tag_name"`
	X               float64 `json:"x"`
	Y               float64 `json:"y"`
	Width           float64 `json:"width"`
	Height          float64 `json:"height"`
	ZIndex          int     `json:"z_index"`
	Visible         bool    `json:"visible"`
	Enabled         bool    `json:"enabled"`
	Focused         bool    `json:"focused"`
	Selected        bool    `json:"selected"`
	Checked         bool    `json:"checked"`
	VisualDominance float64 `json:"visual_dominance"`
	Confidence      float64 `json:"confidence"` // 0-1 alignment confidence
	// T21.1: From accessibility tree
	AXRole           string `json:"ax_role,omitempty"`
	AXName           string `json:"ax_name,omitempty"`
	AXDescription    string `json:"ax_description,omitempty"`
	// T22.2: Visual DNA for self-healing
	FontWeight       int     `json:"font_weight,omitempty"`
	BorderRadius     float64 `json:"border_radius,omitempty"`
	// T22.3: Spatial context for self-healing
	NearbyElements  []string `json:"nearby_elements,omitempty"` // Refs of stable nearby elements
	ProximityScore   float64  `json:"proximity_score,omitempty"`  // Score based on stable neighbors
}

// NewAXAlignment creates a new AX alignment processor
func NewAXAlignment() *AXAlignment {
	return &AXAlignment{
		Elements: make([]*AlignedElement, 0),
	}
}

// MapSpatialMap maps accessibility data to spatial elements using JavaScript
// This is more reliable than CDP for getting AX properties mapped to DOM elements
func (ax *AXAlignment) MapSpatialMap(spatialMap *SpatialMap, page *rod.Page) error {
	ax.mu.Lock()
	defer ax.mu.Unlock()

	// Use JavaScript to get accessibility properties for elements
	jsCode := `
		(function() {
			const results = [];
			const elements = document.querySelectorAll('*');
			
			elements.forEach((el, index) => {
				try {
					// Check if element has accessibility info
					if (el.accessibleName || el.accessibilityDescription || 
					    el.tagName === 'BUTTON' || el.tagName === 'A' || 
					    el.tagName === 'INPUT' || el.tagName === 'SELECT' ||
					    el.getAttribute('role')) {
						
						const ax = el.getAttribute('aria-label') || '';
						const role = el.getAttribute('role') || '';
						
						results.push({
							tag: el.tagName.toLowerCase(),
							text: (el.innerText || '').trim().substring(0, 100),
							placeholder: el.placeholder || '',
							ariaLabel: ax,
							role: role,
							title: el.title || '',
							id: el.id || '',
							classes: el.className || ''
						});
					}
				} catch(e) {}
			});
			
			return JSON.stringify(results);
		})()
	`

	resp, err := proto.RuntimeEvaluate{
		Expression:   jsCode,
		ReturnByValue: true,
	}.Call(page)

	if err != nil || resp == nil || resp.ExceptionDetails != nil {
		return fmt.Errorf("failed to get accessibility data: %v", err)
	}

	if resp.Result.Type != "string" {
		return fmt.Errorf("unexpected result type: %s", resp.Result.Type)
	}

	// Parse accessibility data
	var axData []struct {
		Tag          string `json:"tag"`
		Text         string `json:"text"`
		Placeholder  string `json:"placeholder"`
		AriaLabel    string `json:"ariaLabel"`
		Role         string `json:"role"`
		Title        string `json:"title"`
		ID           string `json:"id"`
		Classes      string `json:"classes"`
	}

	if err := json.Unmarshal([]byte(resp.Result.Value.String()), &axData); err != nil {
		return fmt.Errorf("failed to parse accessibility data: %w", err)
	}

	// Match accessibility data to spatial elements based on position and attributes
	for _, spatialEl := range spatialMap.Elements {
		// Try to match by ID first
		if spatialEl.Attributes != nil {
			if id, ok := spatialEl.Attributes["id"]; ok {
				for _, ax := range axData {
					if ax.ID == id {
						spatialEl.AXRole = ax.Role
						spatialEl.AXName = ax.AriaLabel + ax.Text
						spatialEl.AXDescription = ax.Title
						break
					}
				}
			}
		}
		
		// Also try matching by tag + text
		if spatialEl.AXName == "" {
			for _, ax := range axData {
				if ax.Tag == spatialEl.TagName && 
				   (strings.Contains(spatialEl.Text, ax.Text) || strings.Contains(ax.Text, spatialEl.Text)) {
					spatialEl.AXRole = ax.Role
					spatialEl.AXName = ax.AriaLabel + ax.Text
					spatialEl.AXDescription = ax.Title
					break
				}
			}
		}
	}

	ax.SpatialMap = spatialMap
	return nil
}

// Extract extracts aligned accessibility and spatial data in one call
// This is the main entry point - combines spatial extraction with AX mapping
func (ax *AXAlignment) Extract(page *rod.Page, spatialExtractor *SpatialExtractor) (*AXAlignment, error) {
	ax.mu.Lock()
	defer ax.mu.Unlock()

	// Step 1: Extract spatial data
	spatialMap, err := spatialExtractor.Extract(page)
	if err != nil {
		return nil, fmt.Errorf("failed to extract spatial data: %w", err)
	}

	// Step 2: Map accessibility data to spatial elements (inline logic)
	jsCode := `
		(function() {
			const results = [];
			const elements = document.querySelectorAll('*');
			
			elements.forEach((el, index) => {
				try {
					if (el.accessibleName || el.accessibilityDescription || 
					    el.tagName === 'BUTTON' || el.tagName === 'A' || 
					    el.tagName === 'INPUT' || el.tagName === 'SELECT' ||
					    el.getAttribute('role')) {
						
						const ax = el.getAttribute('aria-label') || '';
						const role = el.getAttribute('role') || '';
						
						results.push({
							tag: el.tagName.toLowerCase(),
							text: (el.innerText || '').trim().substring(0, 100),
							placeholder: el.placeholder || '',
							ariaLabel: ax,
							role: role,
							title: el.title || '',
							id: el.id || '',
							classes: el.className || ''
						});
					}
				} catch(e) {}
			});
			
			return JSON.stringify(results);
		})()
	`

	resp, err := proto.RuntimeEvaluate{
		Expression:   jsCode,
		ReturnByValue: true,
	}.Call(page)

	if err == nil && resp != nil && resp.ExceptionDetails == nil && resp.Result.Type == "string" {
		var axData []struct {
			Tag          string `json:"tag"`
			Text         string `json:"text"`
			Placeholder  string `json:"placeholder"`
			AriaLabel    string `json:"ariaLabel"`
			Role         string `json:"role"`
			Title        string `json:"title"`
			ID           string `json:"id"`
			Classes      string `json:"classes"`
		}

		if jsonErr := json.Unmarshal([]byte(resp.Result.Value.String()), &axData); jsonErr == nil {
			// Match accessibility data to spatial elements
			for _, spatialEl := range spatialMap.Elements {
				if spatialEl.Attributes != nil {
					if id, ok := spatialEl.Attributes["id"]; ok {
						for _, ax := range axData {
							if ax.ID == id {
								spatialEl.AXRole = ax.Role
								spatialEl.AXName = ax.AriaLabel + ax.Text
								spatialEl.AXDescription = ax.Title
								break
							}
						}
					}
				}
				if spatialEl.AXName == "" {
					for _, ax := range axData {
						if ax.Tag == spatialEl.TagName && 
						   (strings.Contains(spatialEl.Text, ax.Text) || strings.Contains(ax.Text, spatialEl.Text)) {
							spatialEl.AXRole = ax.Role
							spatialEl.AXName = ax.AriaLabel + ax.Text
							spatialEl.AXDescription = ax.Title
							break
						}
					}
				}
			}
		}
	}

	ax.SpatialMap = spatialMap

	// Step 3: Build aligned elements with confidence scoring
	ax.Elements = ax.buildAlignedElements(spatialMap)

	return ax, nil
}

func (ax *AXAlignment) buildAlignedElements(spatialMap *SpatialMap) []*AlignedElement {
	elements := make([]*AlignedElement, 0)

	// Create aligned elements from spatial map
	for _, spatialEl := range spatialMap.Elements {
		// Calculate confidence based on what AX data we found
		confidence := 0.3 // Base confidence
		
		if spatialEl.AXRole != "" {
			confidence += 0.2 // Has role
		}
		if spatialEl.AXName != "" {
			confidence += 0.2 // Has name/label
		}
		if spatialEl.AXDescription != "" {
			confidence += 0.1 // Has description
		}
		// Cap at 1.0
		if confidence > 1.0 {
			confidence = 1.0
		}
		
		el := &AlignedElement{
			Ref:             spatialEl.Ref,
			BackendNodeID:   spatialEl.BackendNodeID,
			Role:            spatialEl.AXRole,
			Name:            spatialEl.AXName,
			Description:     spatialEl.AXDescription,
			TagName:         spatialEl.TagName,
			X:               spatialEl.X,
			Y:               spatialEl.Y,
			Width:           spatialEl.Width,
			Height:          spatialEl.Height,
			ZIndex:          spatialEl.ZIndex,
			Visible:         spatialEl.Visible,
			VisualDominance: spatialEl.VisualDominance,
			Confidence:      confidence,
			Enabled:         true,
		}
		elements = append(elements, el)
	}

	// If no spatial elements, create default
	if len(elements) == 0 {
		elements = append(elements, &AlignedElement{
			Ref:        "ax0",
			Role:       "WebArea",
			Name:       "Page",
			TagName:    "body",
			X:          0,
			Y:          0,
			Width:      800,
			Height:     600,
			Visible:    true,
			Confidence: 0.5,
		})
	}

	return elements
}

// FindElementByRole finds elements by accessibility role
func (ax *AXAlignment) FindElementByRole(role string) []*AlignedElement {
	ax.mu.RLock()
	defer ax.mu.RUnlock()

	result := make([]*AlignedElement, 0)
	for _, el := range ax.Elements {
		if el.Role == role {
			result = append(result, el)
		}
	}
	return result
}

// FindElementByName finds elements by accessibility name
func (ax *AXAlignment) FindElementByName(name string) []*AlignedElement {
	ax.mu.RLock()
	defer ax.mu.RUnlock()

	result := make([]*AlignedElement, 0)
	for _, el := range ax.Elements {
		if el.Name == name || containsIgnoreCase(el.Name, name) {
			result = append(result, el)
		}
	}
	return result
}

// FindInteractiveElements finds all interactive elements
func (ax *AXAlignment) FindInteractiveElements() []*AlignedElement {
	ax.mu.RLock()
	defer ax.mu.RUnlock()

	tags := []string{"button", "input", "a", "select", "textarea"}

	result := make([]*AlignedElement, 0)
	for _, el := range ax.Elements {
		for _, tag := range tags {
			if el.TagName == tag {
				result = append(result, el)
				break
			}
		}
	}
	return result
}

// GetConfidenceScore returns the overall alignment confidence
func (ax *AXAlignment) GetConfidenceScore() float64 {
	ax.mu.RLock()
	defer ax.mu.RUnlock()

	if len(ax.Elements) == 0 {
		return 0
	}

	var total float64
	for _, el := range ax.Elements {
		total += el.Confidence
	}
	return total / float64(len(ax.Elements))
}

// CalculateSpatialContext adds proximity tracking for T22.3
// This helps with self-healing when elements move
func (ax *AXAlignment) CalculateSpatialContext() {
	ax.mu.Lock()
	defer ax.mu.Unlock()

	if len(ax.Elements) == 0 {
		return
	}

	// For each element, find nearby stable elements
	for i, el := range ax.Elements {
		// Find elements within 100px that have high confidence
		var nearby []string
		var proximityScore float64
		
		for j, other := range ax.Elements {
			if i == j {
				continue
			}
			
			// Calculate distance
			dx := el.X - other.X
			dy := el.Y - other.Y
			distance := math.Sqrt(dx*dx + dy*dy)
			
			// If within 100px and other element is stable (high confidence)
			if distance < 100 && other.Confidence > 0.5 {
				nearby = append(nearby, other.Ref)
				proximityScore += (1 - distance/100) * other.Confidence
			}
		}
		
		el.NearbyElements = nearby
		el.ProximityScore = proximityScore
	}
}

// ExtractVisualDNA extracts visual characteristics for self-healing (T22.2)
func (ax *AXAlignment) ExtractVisualDNA(page *rod.Page) error {
	jsCode := `
		(function() {
			const results = [];
			const elements = document.querySelectorAll('*');
			
			elements.forEach((el, index) => {
				const rect = el.getBoundingClientRect();
				if (rect.width < 5 || rect.height < 5) return;
				
				const style = window.getComputedStyle(el);
				
				results.push({
					tag: el.tagName.toLowerCase(),
					id: el.id || '',
					fontWeight: parseInt(style.fontWeight) || 400,
					borderRadius: parseFloat(style.borderRadius) || 0,
					textLength: (el.innerText || '').length
				});
			});
			
			return JSON.stringify(results);
		})()
	`

	resp, err := proto.RuntimeEvaluate{
		Expression:   jsCode,
		ReturnByValue: true,
	}.Call(page)

	if err != nil || resp == nil || resp.ExceptionDetails != nil {
		return fmt.Errorf("failed to extract visual DNA: %v", err)
	}

	if resp.Result.Type != "string" {
		return nil // Not critical
	}

	var visualData []struct {
		Tag         string `json:"tag"`
		ID          string `json:"id"`
		FontWeight  int    `json:"fontWeight"`
		BorderRadius float64 `json:"borderRadius"`
		TextLength  int    `json:"textLength"`
	}

	if err := json.Unmarshal([]byte(resp.Result.Value.String()), &visualData); err != nil {
		return nil // Not critical
	}

	// Match visual data to aligned elements
	for _, el := range ax.Elements {
		for _, v := range visualData {
			if v.ID != "" && el.TagName == v.Tag {
				el.FontWeight = v.FontWeight
				el.BorderRadius = v.BorderRadius
				break
			}
		}
	}

	return nil
}
