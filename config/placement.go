package config

import "strings"

// Placement controls where new snippet cards appear.
type Placement string

const (
	PlacementNearSelection Placement = "near-selection"
	PlacementNearCursor    Placement = "near-cursor"
	PlacementFixed         Placement = "fixed"
)

// EffectivePlacement returns the card placement mode (default near-selection).
func (c Config) EffectivePlacement() Placement {
	switch strings.ToLower(strings.TrimSpace(c.Card.Placement)) {
	case "", "near-selection", "near_selection":
		return PlacementNearSelection
	case "near-cursor", "near_cursor":
		return PlacementNearCursor
	case "fixed":
		return PlacementFixed
	default:
		return PlacementNearSelection
	}
}
