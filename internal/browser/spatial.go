package browser

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// SpatialConfig controls spatial extraction behavior
type SpatialConfig struct {
	// Enabled controls whether spatial extraction is available
	// Default: false (lazy mode - must be explicitly called)
	Enabled bool `json:"enabled"`

	// ExtractAttributes controls whether DOM attributes are extracted
	// Default: true - captures id, class, href, src, etc.
	ExtractAttributes bool `json:"extract_attributes"`

	// ExtractText controls whether element text content is extracted
	// Default: true - captures innerText for content elements
	ExtractText bool `json:"extract_text"`

	// MinElementSize filters out tiny elements below this threshold
	// Default: 1 (pixels)
	MinElementSize float64 `json:"min_element_size"`

	// MaxElements limits the number of elements to extract
	// Default: 0 (no limit)
	MaxElements int `json:"max_elements"`
}

// DefaultSpatialConfig returns the default lazy-loading configuration
func DefaultSpatialConfig() *SpatialConfig {
	return &SpatialConfig{
		Enabled:            false, // Lazy - must be explicitly called
		ExtractAttributes:  true,
		ExtractText:        true,
		MinElementSize:     1,
		MaxElements:        0, // No limit
	}
}

// SpatialElement represents spatial data for an element
type SpatialElement struct {
	Ref              string                 `json:"ref"`
	BackendNodeID    int                    `json:"backend_node_id,omitempty"` // Mapped from accessibility tree
	X                float64                `json:"x"`
	Y                float64                `json:"y"`
	Width            float64                `json:"width"`
	Height           float64                `json:"height"`
	CenterX          float64                `json:"center_x"`
	CenterY          float64                `json:"center_y"`
	ZIndex           int                    `json:"z_index"`
	Visible          bool                   `json:"visible"`
	Color            string                 `json:"color,omitempty"`
	BackgroundColor  string                 `json:"background_color,omitempty"`
	FontSize         float64                `json:"font_size,omitempty"`
	TagName          string                 `json:"tag_name"`
	Text             string                 `json:"text,omitempty"`            // innerText content
	InputValue       string                 `json:"input_value,omitempty"`    // For input/textarea
	Attributes       map[string]string      `json:"attributes,omitempty"`     // id, class, href, etc.
	Children         []string                `json:"children,omitempty"`      // Child element refs
	Parent           string                 `json:"parent,omitempty"`         // Parent element ref
	VisualDominance  float64                `json:"visual_dominance"`
	// T21.1: Accessibility mapping fields
	AXRole           string                 `json:"ax_role,omitempty"`        // From accessibility tree
	AXName           string                 `json:"ax_name,omitempty"`       // From accessibility tree
	AXDescription    string                 `json:"ax_description,omitempty"` // From accessibility tree
	// T22.2: Visual DNA for self-healing locators
	FontWeight       int                    `json:"font_weight,omitempty"`    // Font weight (100-900)
	BorderRadius     float64                `json:"border_radius,omitempty"` // Border radius
	IconSVGPath      string                 `json:"icon_svg_path,omitempty"`  // SVG path if icon element
}

// SpatialMap represents a spatial snapshot of the page
type SpatialMap struct {
	URL        string            `json:"url"`
	Width      float64           `json:"width"`
	Height     float64           `json:"height"`
	Elements   []*SpatialElement `json:"elements"`
	Timestamp  string            `json:"timestamp"`
	TotalArea  float64           `json:"total_area"`
	PageTitle  string            `json:"page_title,omitempty"`
	ExtractMs  int64             `json:"extract_ms"` // Time taken to extract
}

// SpatialRelation represents the relationship between two elements
type SpatialRelation string

const (
	SpatialRelationNone     SpatialRelation = "none"
	SpatialRelationAbove   SpatialRelation = "above"
	SpatialRelationBelow   SpatialRelation = "below"
	SpatialRelationLeft    SpatialRelation = "left"
	SpatialRelationRight   SpatialRelation = "right"
	SpatialRelationInside  SpatialRelation = "inside"
	SpatialRelationOverlaps SpatialRelation = "overlaps"
)

// SpatialExtractor extracts spatial data from a page
type SpatialExtractor struct {
	config *SpatialConfig
}

// NewSpatialExtractor creates a new spatial extractor with default config
func NewSpatialExtractor() *SpatialExtractor {
	return &SpatialExtractor{
		config: DefaultSpatialConfig(),
	}
}

// NewSpatialExtractorWithConfig creates a spatial extractor with custom config
func NewSpatialExtractorWithConfig(config *SpatialConfig) *SpatialExtractor {
	if config == nil {
		config = DefaultSpatialConfig()
	}
	return &SpatialExtractor{
		config: config,
	}
}

// Extract extracts spatial data from a page (only if Enabled is true)
func (se *SpatialExtractor) Extract(page *rod.Page) (*SpatialMap, error) {
	startTime := time.Now()

	// Get page info
	pageInfo, err := page.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get page info: %w", err)
	}

	// Get page title via CDP
	pageTitle := ""
	titleResp, _ := proto.DOMGetDocument{}.Call(page)
	if titleResp != nil && titleResp.Root != nil {
		titleJS := "document.title"
		titleResult, _ := proto.RuntimeEvaluate{Expression: titleJS, ReturnByValue: true}.Call(page)
		if titleResult != nil && titleResult.Result.Type == "string" {
			pageTitle = titleResult.Result.Value.String()
		}
	}

	// Use default viewport
	var (
		width  float64 = 1280
		height float64 = 800
	)

	// Try to get viewport via CDP
	viewportJS := "(window.innerWidth || 1280) + ',' + (window.innerHeight || 800)"
	viewportResult, _ := proto.RuntimeEvaluate{Expression: viewportJS, ReturnByValue: true}.Call(page)
	if viewportResult != nil && viewportResult.Result.Type == "string" {
		v := viewportResult.Result.Value.String()
		fmt.Sscanf(v, "%f,%f", &width, &height)
	}

	spatialMap := &SpatialMap{
		URL:       pageInfo.URL,
		Width:     width,
		Height:    height,
		Elements:  make([]*SpatialElement, 0),
		Timestamp: time.Now().Format(time.RFC3339),
		PageTitle: pageTitle,
	}

	// Extract elements via CDP JavaScript
	elements := se.extractViaCDP(page)
	if len(elements) == 0 {
		elements = se.getBasicElements()
	}

	spatialMap.Elements = elements
	spatialMap.ExtractMs = time.Since(startTime).Milliseconds()

	// Calculate total area and visual dominance
	var totalArea float64
	for _, el := range elements {
		totalArea += el.Width * el.Height
	}
	spatialMap.TotalArea = totalArea

	// Calculate visual dominance for each element
	for _, el := range elements {
		el.VisualDominance = se.calculateVisualDominance(el, totalArea)
	}

	return spatialMap, nil
}

// extractViaCDP uses CDP Runtime.evaluate to get element geometry and properties
func (se *SpatialExtractor) extractViaCDP(page *rod.Page) []*SpatialElement {
	// Build JavaScript based on config
	extractAttrs := se.config.ExtractAttributes
	extractText := se.config.ExtractText
	minSize := se.config.MinElementSize
	maxElements := se.config.MaxElements

	// Build the extraction JS
	jsCode := fmt.Sprintf(`
		(function() {
			const elements = [];
			const allElements = document.querySelectorAll('*');
			
			// Attributes to extract (safe, non-sensitive)
			const safeAttrs = ['id', 'class', 'name', 'type', 'href', 'src', 'alt', 'title', 'placeholder', 'role', 'aria-label', 'data-testid', 'data-cy'];
			
			const maxElements = %d;
			const minSize = %f;
			
			allElements.forEach((el, index) => {
				if (maxElements > 0 && index >= maxElements) return;
				
				const rect = el.getBoundingClientRect();
				const style = window.getComputedStyle(el);
				
				// Only include visible elements with meaningful size
				if (rect.width > minSize && rect.height > minSize && 
					style.display !== 'none' && 
					style.visibility !== 'hidden' &&
					style.opacity !== '0') {
					
					const element = {
						ref: 'sp' + index,
						tagName: el.tagName.toLowerCase(),
						x: rect.x,
						y: rect.y,
						width: rect.width,
						height: rect.height,
						centerX: rect.x + rect.width / 2,
						centerY: rect.y + rect.height / 2,
						visible: true,
						color: style.color,
						backgroundColor: style.backgroundColor,
						fontSize: parseFloat(style.fontSize) || 0,
						zIndex: parseInt(style.zIndex) || 0
					};
					
					// Extract text content for content elements
					%s
					
					// Extract safe attributes
					%s
					
					elements.push(element);
				}
			});
			
			return JSON.stringify(elements);
		})()
	`, maxElements, minSize,
		se.ternary(extractText, `if (['DIV', 'SPAN', 'P', 'A', 'BUTTON', 'LI', 'H1', 'H2', 'H3', 'H4', 'H5', 'H6', 'LABEL', 'TD', 'TH'].includes(el.tagName)) { element.text = (el.innerText || '').trim().substring(0, 500); } else if (['INPUT', 'TEXTAREA'].includes(el.tagName)) { element.inputValue = (el.value || '').trim().substring(0, 500); }`, ""),
		se.ternary(extractAttrs, `element.attributes = {}; safeAttrs.forEach(attr => { if (el.hasAttribute(attr)) { element.attributes[attr] = el.getAttribute(attr); } });`, ""))

	// Use CDP directly
	resp, err := proto.RuntimeEvaluate{
		Expression:   jsCode,
		ReturnByValue: true,
	}.Call(page)

	if err != nil || resp == nil || resp.ExceptionDetails != nil {
		if resp != nil && resp.ExceptionDetails != nil {
			fmt.Printf("CDP Exception: %s\n", resp.ExceptionDetails.Exception.Description)
		}
		return nil
	}

	// Get the result
	result := resp.Result
	if result.Type != "string" {
		return nil
	}

	elementsJSON := result.Value.String()
	if err != nil || elementsJSON == "" {
		return nil
	}

	// Parse the JSON
	var rawElements []map[string]interface{}
	if err := json.Unmarshal([]byte(elementsJSON), &rawElements); err != nil {
		return nil
	}

	elements := make([]*SpatialElement, 0, len(rawElements))
	for _, raw := range rawElements {
		el := &SpatialElement{
			Attributes: make(map[string]string),
		}

		if v, ok := raw["ref"].(string); ok {
			el.Ref = v
		}
		if v, ok := raw["tagName"].(string); ok {
			el.TagName = v
		}
		if v, ok := raw["x"].(float64); ok {
			el.X = v
		}
		if v, ok := raw["y"].(float64); ok {
			el.Y = v
		}
		if v, ok := raw["width"].(float64); ok {
			el.Width = v
		}
		if v, ok := raw["height"].(float64); ok {
			el.Height = v
		}
		if v, ok := raw["centerX"].(float64); ok {
			el.CenterX = v
		}
		if v, ok := raw["centerY"].(float64); ok {
			el.CenterY = v
		}
		if v, ok := raw["visible"].(bool); ok {
			el.Visible = v
		}
		if v, ok := raw["color"].(string); ok {
			el.Color = v
		}
		if v, ok := raw["backgroundColor"].(string); ok {
			el.BackgroundColor = v
		}
		if v, ok := raw["fontSize"].(float64); ok {
			el.FontSize = v
		}
		if v, ok := raw["zIndex"].(float64); ok {
			el.ZIndex = int(v)
		}
		if v, ok := raw["text"].(string); ok {
			el.Text = v
		}
		if v, ok := raw["inputValue"].(string); ok {
			el.InputValue = v
		}
		if attrs, ok := raw["attributes"].(map[string]interface{}); ok && attrs != nil {
			for k, vv := range attrs {
				if sv, ok := vv.(string); ok {
					el.Attributes[k] = sv
				}
			}
		}

		elements = append(elements, el)
	}

	return elements
}

// ternary is a helper for conditional JS generation
func (se *SpatialExtractor) ternary(condition bool, trueVal, falseVal string) string {
	if condition {
		return trueVal
	}
	return falseVal
}

// getBasicElements returns basic page elements
func (se *SpatialExtractor) getBasicElements() []*SpatialElement {
	return []*SpatialElement{
		{
			Ref:     "sp0",
			TagName: "body",
			X:       0,
			Y:       0,
			Width:   800,
			Height:  600,
			Visible: true,
		},
	}
}

// calculateVisualDominance calculates how visually dominant an element is
func (se *SpatialExtractor) calculateVisualDominance(el *SpatialElement, totalArea float64) float64 {
	area := el.Width * el.Height

	areaScore := 0.0
	if totalArea > 0 {
		areaScore = (area / totalArea) * 10
	}

	contentBonus := 0.0
	switch el.TagName {
	case "img", "video", "canvas", "figure":
		contentBonus = 0.3
	case "button", "a", "input":
		contentBonus = 0.2
	case "h1", "h2", "h3", "h4", "h5", "h6":
		contentBonus = 0.15
	}

	// Bonus for having text content
	if el.Text != "" {
		contentBonus += 0.1
	}

	// Bonus for interactive attributes
	if _, ok := el.Attributes["href"]; ok {
		contentBonus += 0.05
	}
	if _, ok := el.Attributes["onclick"]; ok {
		contentBonus += 0.1
	}

	if el.BackgroundColor != "" {
		contentBonus += 0.1
	}

	score := areaScore + contentBonus
	if score > 1 {
		score = 1
	}

	return math.Round(score*100) / 100
}

// FindElementByPosition finds an element at a specific position
func (se *SpatialExtractor) FindElementByPosition(spatialMap *SpatialMap, x, y float64) *SpatialElement {
	sorted := make([]*SpatialElement, len(spatialMap.Elements))
	copy(sorted, spatialMap.Elements)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ZIndex > sorted[j].ZIndex
	})

	for _, el := range sorted {
		if !el.Visible {
			continue
		}
		if x >= el.X && x <= el.X+el.Width &&
			y >= el.Y && y <= el.Y+el.Height {
			return el
		}
	}

	return nil
}

// FindElementsByText finds elements containing the specified text
func (se *SpatialExtractor) FindElementsByText(spatialMap *SpatialMap, text string) []*SpatialElement {
	var results []*SpatialElement
	
	for _, el := range spatialMap.Elements {
		if !el.Visible {
			continue
		}
		if el.Text == "" {
			continue
		}
		elTextLower := el.Text
		if len(elTextLower) > 100 {
			elTextLower = elTextLower[:100]
		}
		if containsIgnoreCase(el.Text, text) || containsIgnoreCase(elTextLower, text) {
			results = append(results, el)
		}
	}
	
	return results
}

// FindElementByAttribute finds elements with matching attribute
func (se *SpatialExtractor) FindElementByAttribute(spatialMap *SpatialMap, attr, value string) []*SpatialElement {
	var results []*SpatialElement
	
	for _, el := range spatialMap.Elements {
		if !el.Visible {
			continue
		}
		if el.Attributes == nil {
			continue
		}
		if attrVal, ok := el.Attributes[attr]; ok && containsIgnoreCase(attrVal, value) {
			results = append(results, el)
		}
	}
	
	return results
}

// GetSpatialRelation determines the spatial relationship between two elements
func (se *SpatialExtractor) GetSpatialRelation(el1, el2 *SpatialElement) SpatialRelation {
	if el1.X >= el2.X && el1.X+el1.Width <= el2.X+el2.Width &&
		el1.Y >= el2.Y && el1.Y+el1.Height <= el2.Y+el2.Height {
		return SpatialRelationInside
	}

	if el1.X+el1.Width < el2.X {
		if el1.Y < el2.Y+el2.Height && el1.Y+el1.Height > el2.Y {
			return SpatialRelationLeft
		}
	}

	if el1.X > el2.X+el2.Width {
		if el1.Y < el2.Y+el2.Height && el1.Y+el1.Height > el2.Y {
			return SpatialRelationRight
		}
	}

	if el1.Y+el1.Height < el2.Y {
		if el1.X < el2.X+el2.Width && el1.X+el1.Width > el2.X {
			return SpatialRelationAbove
		}
	}

	if el1.Y > el2.Y+el2.Height {
		if el1.X < el2.X+el2.Width && el1.X+el1.Width > el2.X {
			return SpatialRelationBelow
		}
	}

	if se.elementsOverlap(el1, el2) {
		return SpatialRelationOverlaps
	}

	return SpatialRelationNone
}

func (se *SpatialExtractor) elementsOverlap(el1, el2 *SpatialElement) bool {
	return el1.X < el2.X+el2.Width &&
		el1.X+el1.Width > el2.X &&
		el1.Y < el2.Y+el2.Height &&
		el1.Y+el1.Height > el2.Y
}

func (se *SpatialExtractor) Distance(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	return math.Sqrt(dx*dx + dy*dy)
}

func (se *SpatialExtractor) FindNearestElement(spatialMap *SpatialMap, x, y float64, limit int) []*SpatialElement {
	type distanceElement struct {
		el       *SpatialElement
		distance float64
	}

	distances := make([]distanceElement, 0)
	for _, el := range spatialMap.Elements {
		if !el.Visible {
			continue
		}
		d := se.Distance(x, y, el.CenterX, el.CenterY)
		distances = append(distances, distanceElement{el: el, distance: d})
	}

	sort.Slice(distances, func(i, j int) bool {
		return distances[i].distance < distances[j].distance
	})

	result := make([]*SpatialElement, 0)
	for i := 0; i < limit && i < len(distances); i++ {
		result = append(result, distances[i].el)
	}

	return result
}

func (se *SpatialExtractor) FindElementsAbove(spatialMap *SpatialMap, y float64, limit int) []*SpatialElement {
	var above []*SpatialElement
	for _, el := range spatialMap.Elements {
		if el.Visible && el.Y+el.Height < y {
			above = append(above, el)
		}
	}

	sort.Slice(above, func(i, j int) bool {
		return (above[i].Y + above[i].Height) > (above[j].Y + above[j].Height)
	})

	if len(above) > limit {
		above = above[:limit]
	}
	return above
}

func (se *SpatialExtractor) FindElementsBelow(spatialMap *SpatialMap, y float64, limit int) []*SpatialElement {
	var below []*SpatialElement
	for _, el := range spatialMap.Elements {
		if el.Visible && el.Y > y {
			below = append(below, el)
		}
	}

	sort.Slice(below, func(i, j int) bool {
		return below[i].Y < below[j].Y
	})

	if len(below) > limit {
		below = below[:limit]
	}
	return below
}

// Helper function for case-insensitive string matching (uses captcha.go version)
