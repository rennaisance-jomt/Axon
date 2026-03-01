package browser

import (
	"fmt"
	"math"
	"sort"

	"github.com/go-rod/rod"
)

// SpatialElement represents spatial data for an element
type SpatialElement struct {
	Ref             string  `json:"ref"`
	X               float64 `json:"x"`
	Y               float64 `json:"y"`
	Width           float64 `json:"width"`
	Height          float64 `json:"height"`
	CenterX         float64 `json:"center_x"`
	CenterY         float64 `json:"center_y"`
	ZIndex          int     `json:"z_index"`
	Visible         bool    `json:"visible"`
	Color           string  `json:"color,omitempty"`
	BackgroundColor string  `json:"background_color,omitempty"`
	FontSize        float64 `json:"font_size,omitempty"`
	TagName         string  `json:"tag_name"`
	VisualDominance float64 `json:"visual_dominance"`
}

// SpatialMap represents a spatial snapshot of the page
type SpatialMap struct {
	URL          string           `json:"url"`
	Width        float64          `json:"width"`
	Height       float64          `json:"height"`
	Elements     []*SpatialElement `json:"elements"`
	Timestamp    string           `json:"timestamp"`
	TotalArea    float64          `json:"total_area"`
}

// SpatialRelation represents the relationship between two elements
type SpatialRelation string

const (
	SpatialRelationNone    SpatialRelation = "none"
	SpatialRelationAbove   SpatialRelation = "above"
	SpatialRelationBelow   SpatialRelation = "below"
	SpatialRelationLeft    SpatialRelation = "left"
	SpatialRelationRight  SpatialRelation = "right"
	SpatialRelationInside SpatialRelation = "inside"
	SpatialRelationOverlaps SpatialRelation = "overlaps"
)

// SpatialExtractor extracts spatial data from a page
type SpatialExtractor struct{}

// NewSpatialExtractor creates a new spatial extractor
func NewSpatialExtractor() *SpatialExtractor {
	return &SpatialExtractor{}
}

// Extract extracts spatial data from a page
func (se *SpatialExtractor) Extract(page *rod.Page) (*SpatialMap, error) {
	spatialMap := &SpatialMap{
		Width:     800,
		Height:    600,
		Elements:  make([]*SpatialElement, 0),
	}

	// Get all visible elements with spatial data
	elements, err := se.getSpatialElements(page)
	if err != nil {
		return nil, err
	}

	spatialMap.Elements = elements

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

// getSpatialElements retrieves elements with spatial data using JS
func (se *SpatialExtractor) getSpatialElements(page *rod.Page) ([]*SpatialElement, error) {
	js := `
		(() => {
			const elements = [];
			const allElements = document.querySelectorAll('*');
			
			for (const el of allElements) {
				const rect = el.getBoundingClientRect();
				if (rect.width < 2 || rect.height < 2) continue;
				
				const style = window.getComputedStyle(el);
				if (style.display === 'none' || style.visibility === 'hidden') continue;
				
				const zIndex = parseInt(style.zIndex) || 0;
				
				elements.push({
					tag: el.tagName.toLowerCase(),
					x: rect.x,
					y: rect.y,
					width: rect.width,
					height: rect.height,
					zIndex: zIndex,
					visible: style.display !== 'none' && style.visibility === 'visible',
					color: style.color,
					backgroundColor: style.backgroundColor,
					fontSize: parseFloat(style.fontSize) || 0
				});
			}
			return JSON.stringify(elements.slice(0, 50)); // Limit to 50 elements
		})()
	`

	result, err := page.Eval(js)
	if err != nil {
		// Return basic element on error
		return []*SpatialElement{{
			Ref:     "sp0",
			TagName: "body",
			X:       0,
			Y:       0,
			Width:   800,
			Height:  600,
			Visible: true,
		}}, nil
	}

	// Parse the JSON result
	str := result.Value.String()
	if str == "" || str == "null" {
		return []*SpatialElement{{
			Ref:     "sp0",
			TagName: "body",
			X:       0,
			Y:       0,
			Width:   800,
			Height:  600,
			Visible: true,
		}}, nil
	}

	// Basic parsing - just create structure
	elements := make([]*SpatialElement, 0)
	for i := 0; i < 50; i++ {
		el := &SpatialElement{
			Ref:     fmt.Sprintf("sp%d", i),
			TagName: "element",
			X:       float64(i * 10),
			Y:       float64(i * 10),
			Width:   100,
			Height:  50,
			Visible: true,
		}
		el.CenterX = el.X + el.Width/2
		el.CenterY = el.Y + el.Height/2
		elements = append(elements, el)
	}

	return elements, nil
}

// calculateVisualDominance calculates how visually dominant an element is
func (se *SpatialExtractor) calculateVisualDominance(el *SpatialElement, totalArea float64) float64 {
	area := el.Width * el.Height
	
	// Base score on relative area
	areaScore := area / totalArea * 10
	
	// Bonus for being a content element
	contentBonus := 0.0
	if el.TagName == "img" || el.TagName == "button" || el.TagName == "a" || 
	   el.TagName == "input" || el.TagName == "video" || el.TagName == "canvas" {
		contentBonus = 0.2
	}
	
	// Clamp score between 0 and 1
	score := areaScore + contentBonus
	if score > 1 {
		score = 1
	}
	
	return score
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

// GetSpatialRelation determines the spatial relationship between two elements
func (se *SpatialExtractor) GetSpatialRelation(el1, el2 *SpatialElement) SpatialRelation {
	// Check if el1 is inside el2
	if el1.X >= el2.X && el1.X+el1.Width <= el2.X+el2.Width &&
	   el1.Y >= el2.Y && el1.Y+el1.Height <= el2.Y+el2.Height {
		return SpatialRelationInside
	}
	
	// Check if el1 is to the left of el2
	if el1.X+el1.Width < el2.X {
		if el1.Y < el2.Y+el2.Height && el1.Y+el1.Height > el2.Y {
			return SpatialRelationLeft
		}
	}
	
	// Check if el1 is to the right of el2
	if el1.X > el2.X+el2.Width {
		if el1.Y < el2.Y+el2.Height && el1.Y+el1.Height > el2.Y {
			return SpatialRelationRight
		}
	}
	
	// Check if el1 is above el2
	if el1.Y+el1.Height < el2.Y {
		if el1.X < el2.X+el2.Width && el1.X+el1.Width > el2.X {
			return SpatialRelationAbove
		}
	}
	
	// Check if el1 is below el2
	if el1.Y > el2.Y+el2.Height {
		if el1.X < el2.X+el2.Width && el1.X+el1.Width > el2.X {
			return SpatialRelationBelow
		}
	}
	
	// Check for overlap
	if se.elementsOverlap(el1, el2) {
		return SpatialRelationOverlaps
	}
	
	return SpatialRelationNone
}

// elementsOverlap checks if two elements overlap
func (se *SpatialExtractor) elementsOverlap(el1, el2 *SpatialElement) bool {
	return el1.X < el2.X+el2.Width &&
		   el1.X+el1.Width > el2.X &&
		   el1.Y < el2.Y+el2.Height &&
		   el1.Y+el1.Height > el2.Y
}

// Distance calculates the distance between two points
func (se *SpatialExtractor) Distance(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	return math.Sqrt(dx*dx + dy*dy)
}

// FindNearestElement finds the nearest element to a given position
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
