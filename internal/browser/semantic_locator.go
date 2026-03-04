package browser

import (
	"regexp"
	"strings"
	"sync"
)

// SemanticLocator provides self-healing element resolution
type SemanticLocator struct {
	mu       sync.RWMutex
	Strategy string
	Anchors  []*Anchor
}

// Anchor provides stable element identification
type Anchor struct {
	Type      string  `json:"type"` // text, role, aria, id, class
	Value     string  `json:"value"`
	Weight    float64 `json:"weight"` // 0-1 importance
	Ordinal   int     `json:"ordinal"` // position among siblings
	Reference string  `json:"reference"` // reference to another element
}

// ResolutionResult contains resolved element data
type ResolutionResult struct {
	Element        *AlignedElement `json:"element"`
	LocatorUsed    string          `json:"locator_used"`
	Confidence     float64         `json:"confidence"`
	Alternatives   []string        `json:"alternatives"`
	HealedFrom     string          `json:"healed_from"` // original locator that failed
	RetryCount     int             `json:"retry_count"`
}

// NewSemanticLocator creates a new semantic locator
func NewSemanticLocator(strategy string) *SemanticLocator {
	return &SemanticLocator{
		Strategy: strategy,
		Anchors:  make([]*Anchor, 0),
	}
}

// AddAnchor adds an anchor to the locator
func (sl *SemanticLocator) AddAnchor(anchorType, value string, weight float64) *SemanticLocator {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	sl.Anchors = append(sl.Anchors, &Anchor{
		Type:   anchorType,
		Value:  value,
		Weight: weight,
	})
	return sl
}

// SetOrdinal sets the ordinal position
func (sl *SemanticLocator) SetOrdinal(ordinal int) *SemanticLocator {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	if len(sl.Anchors) > 0 {
		sl.Anchors[len(sl.Anchors)-1].Ordinal = ordinal
	}
	return sl
}

// Resolve resolves the element using semantic locators
func (sl *SemanticLocator) Resolve(ax *AXAlignment) *ResolutionResult {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	if len(sl.Anchors) == 0 {
		return &ResolutionResult{
			Confidence:  0,
			Alternatives: []string{},
		}
	}

	// Try primary resolution
	result := sl.resolveWithAnchors(ax, 0)

	if result != nil && result.Confidence > 0.5 {
		return result
	}

	// Try healing strategies
	return sl.heal(ax, result)
}

func (sl *SemanticLocator) resolveWithAnchors(ax *AXAlignment, startIdx int) *ResolutionResult {
	ax.mu.RLock()
	defer ax.mu.RUnlock()

	if len(ax.Elements) == 0 {
		return nil
	}

	bestMatch := (*AlignedElement)(nil)
	var bestScore float64
	var alternatives []string

	for _, el := range ax.Elements {
		score := sl.calculateMatchScore(el, startIdx)
		if score > bestScore {
			bestScore = score
			bestMatch = el
			alternatives = append(alternatives, el.Ref)
		}
	}

	if bestMatch == nil {
		return nil
	}

	return &ResolutionResult{
		Element:      bestMatch,
		LocatorUsed:  sl.buildLocatorString(),
		Confidence:   bestScore,
		Alternatives: alternatives,
		RetryCount:   0,
	}
}

func (sl *SemanticLocator) calculateMatchScore(el *AlignedElement, startIdx int) float64 {
	var totalWeight, matchedWeight float64

	for i := startIdx; i < len(sl.Anchors); i++ {
		anchor := sl.Anchors[i]
		totalWeight += anchor.Weight

		match := false
		switch anchor.Type {
		case "text":
			match = containsIgnoreCase(el.Name, anchor.Value) ||
				containsIgnoreCase(el.Description, anchor.Value)
		case "role":
			match = el.Role == anchor.Value
		case "aria":
			match = el.Name == anchor.Value || el.Value == anchor.Value
		case "id":
			match = el.TagName == anchor.Value
		case "class":
			match = false // Would need class info
		case "tag":
			match = el.TagName == anchor.Value
		}

		// Check ordinal if specified
		if anchor.Ordinal > 0 && !match {
			// Could check sibling position
			match = true // Simplified
		}

		if match {
			matchedWeight += anchor.Weight
		}
	}

	if totalWeight == 0 {
		return 0
	}

	return matchedWeight / totalWeight
}

func (sl *SemanticLocator) heal(ax *AXAlignment, result *ResolutionResult) *ResolutionResult {
	if result == nil {
		result = &ResolutionResult{Alternatives: []string{}}
	}

	// Strategy 1: Relax constraints
	result.HealedFrom = sl.buildLocatorString()
	result.RetryCount = 1

	// Try with fewer anchors
	for i := 0; i < len(sl.Anchors)-1; i++ {
		healed := sl.resolveWithAnchors(ax, i)
		if healed != nil && healed.Confidence > 0.3 {
			healed.HealedFrom = result.HealedFrom
			healed.RetryCount = i + 1
			return healed
		}
	}

	// Strategy 2: Fuzzy text match
	healed := sl.fuzzyResolve(ax)
	if healed != nil {
		healed.HealedFrom = result.HealedFrom
		healed.RetryCount = result.RetryCount + 2
		return healed
	}

	// Strategy 3: Position-based fallback
	healed = sl.positionFallback(ax)
	if healed != nil {
		healed.HealedFrom = result.HealedFrom
		healed.RetryCount = result.RetryCount + 3
		return healed
	}

	// T22.3: Strategy 4 - Proximity-based healing using spatial context
	healed = sl.proximityHeal(ax)
	if healed != nil {
		healed.HealedFrom = result.HealedFrom
		healed.RetryCount = result.RetryCount + 4
		return healed
	}

	return result
}

// proximityHeal uses spatial context (nearby stable elements) to find the element
func (sl *SemanticLocator) proximityHeal(ax *AXAlignment) *ResolutionResult {
	ax.mu.RLock()
	defer ax.mu.RUnlock()

	// Get any anchor we can use as reference
	var targetAnchor *Anchor
	for _, a := range sl.Anchors {
		if a.Type == "text" || a.Type == "role" {
			targetAnchor = a
			break
		}
	}

	if targetAnchor == nil {
		return nil
	}

	// Find elements matching the anchor
	var candidates []*AlignedElement
	for _, el := range ax.Elements {
		if targetAnchor.Type == "text" {
			if containsIgnoreCase(el.Name, targetAnchor.Value) || 
			   containsIgnoreCase(el.AXName, targetAnchor.Value) {
				candidates = append(candidates, el)
			}
		} else if targetAnchor.Type == "role" {
			if el.Role == targetAnchor.Value || el.AXRole == targetAnchor.Value {
				candidates = append(candidates, el)
			}
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// Pick the one with highest proximity score (most stable neighbors)
	var best *AlignedElement
	var bestScore float64
	for _, c := range candidates {
		if c.ProximityScore > bestScore {
			bestScore = c.ProximityScore
			best = c
		}
	}

	if best != nil && bestScore > 0 {
		return &ResolutionResult{
			Element:      best,
			LocatorUsed:  "proximity:" + sl.buildLocatorString(),
			Confidence:   0.6, // Reasonable confidence with stable neighbors
			Alternatives: []string{},
		}
	}

	return nil
}

func (sl *SemanticLocator) fuzzyResolve(ax *AXAlignment) *ResolutionResult {
	ax.mu.RLock()
	defer ax.mu.RUnlock()

	// Get target anchor value
	var targetValue string
	for _, anchor := range sl.Anchors {
		if anchor.Type == "text" {
			targetValue = anchor.Value
			break
		}
	}

	if targetValue == "" {
		return nil
	}

	var bestMatch *AlignedElement
	var bestScore float64

	for _, el := range ax.Elements {
		score := fuzzyMatch(targetValue, el.Name)
		if score > bestScore && score > 0.6 {
			bestScore = score
			bestMatch = el
		}
	}

	if bestMatch == nil {
		return nil
	}

	return &ResolutionResult{
		Element:      bestMatch,
		LocatorUsed:  "fuzzy:" + targetValue,
		Confidence:   bestScore,
		Alternatives: []string{},
	}
}

func (sl *SemanticLocator) positionFallback(ax *AXAlignment) *ResolutionResult {
	ax.mu.RLock()
	defer ax.mu.RUnlock()

	// Get first anchor
	if len(sl.Anchors) == 0 {
		return nil
	}

	anchor := sl.Anchors[0]
	if anchor.Type != "role" && anchor.Type != "tag" {
		return nil
	}

	// Find first element matching type
	for _, el := range ax.Elements {
		if el.Role == anchor.Value || el.TagName == anchor.Value {
			return &ResolutionResult{
				Element:      el,
				LocatorUsed:  "position:" + anchor.Value,
				Confidence:   0.3,
				Alternatives: []string{},
			}
		}
	}

	return nil
}

func (sl *SemanticLocator) buildLocatorString() string {
	var parts []string
	for _, anchor := range sl.Anchors {
		parts = append(parts, anchor.Type+":"+anchor.Value)
	}
	return strings.Join(parts, ">>")
}

// FromText creates a locator from text content
func FromText(text string) *SemanticLocator {
	return NewSemanticLocator("text").AddAnchor("text", text, 1.0)
}

// FromRole creates a locator from role
func FromRole(role string) *SemanticLocator {
	return NewSemanticLocator("role").AddAnchor("role", role, 0.8)
}

// FromAria creates a locator from ARIA label
func FromAria(label string) *SemanticLocator {
	return NewSemanticLocator("aria").AddAnchor("aria", label, 1.0)
}

// Combine creates a combined locator
func Combine(locators ...*SemanticLocator) *SemanticLocator {
	combined := NewSemanticLocator("combined")
	for _, loc := range locators {
		loc.mu.RLock()
		combined.Anchors = append(combined.Anchors, loc.Anchors...)
		loc.mu.RUnlock()
	}
	return combined
}

func fuzzyMatch(target, candidate string) float64 {
	if target == "" || candidate == "" {
		return 0
	}

	target = strings.ToLower(target)
	candidate = strings.ToLower(candidate)

	// Exact match
	if target == candidate {
		return 1.0
	}

	// Contains
	if strings.Contains(candidate, target) {
		return 0.9
	}

	// Word overlap
	targetWords := strings.Fields(target)
	candidateWords := strings.Fields(candidate)

	var matchCount int
	for _, tw := range targetWords {
		for _, cw := range candidateWords {
			if tw == cw {
				matchCount++
				break
			}
		}
	}

	if len(targetWords) == 0 {
		return 0
	}

	return float64(matchCount) / float64(len(targetWords))
}

// RegexLocator uses regex patterns for matching
type RegexLocator struct {
	mu      sync.RWMutex
	Pattern string
	regex   *regexp.Regexp
}

// NewRegexLocator creates a new regex locator
func NewRegexLocator(pattern string) (*RegexLocator, error) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	return &RegexLocator{
		Pattern: pattern,
		regex:   regex,
	}, nil
}

// Match checks if element matches the regex
func (rl *RegexLocator) Match(el *AlignedElement) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return rl.regex.MatchString(el.Name) ||
		rl.regex.MatchString(el.Description) ||
		rl.regex.MatchString(el.Value)
}
