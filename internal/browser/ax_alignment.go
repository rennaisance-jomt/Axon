package browser

import (
	"sync"

	"github.com/go-rod/rod"
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
}

// NewAXAlignment creates a new AX alignment processor
func NewAXAlignment() *AXAlignment {
	return &AXAlignment{
		Elements: make([]*AlignedElement, 0),
	}
}

// Extract extracts aligned accessibility and spatial data
func (ax *AXAlignment) Extract(page *rod.Page, spatialMap *SpatialMap) (*AXAlignment, error) {
	ax.mu.Lock()
	defer ax.mu.Unlock()

	ax.SpatialMap = spatialMap

	// Build aligned elements from spatial map
	ax.Elements = ax.buildAlignedElements(spatialMap)

	return ax, nil
}

func (ax *AXAlignment) buildAlignedElements(spatialMap *SpatialMap) []*AlignedElement {
	elements := make([]*AlignedElement, 0)

	// Create aligned elements from spatial map
	for _, spatialEl := range spatialMap.Elements {
		el := &AlignedElement{
			Ref:             spatialEl.Ref,
			TagName:         spatialEl.TagName,
			X:               spatialEl.X,
			Y:               spatialEl.Y,
			Width:           spatialEl.Width,
			Height:          spatialEl.Height,
			ZIndex:          spatialEl.ZIndex,
			Visible:         spatialEl.Visible,
			VisualDominance: spatialEl.VisualDominance,
			Confidence:      0.3,
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
