package config

import "strings"

const (
	WhichKeyTopLeft     = "top-left"
	WhichKeyTopRight    = "top-right"
	WhichKeyBottomLeft  = "bottom-left"
	WhichKeyBottomRight = "bottom-right"
)

// EffectiveWhichKeyCorner returns a known corner, or bottom-right when unset
// or unrecognized.
func (l Leader) EffectiveWhichKeyCorner() string {
	switch strings.ToLower(strings.TrimSpace(l.WhichKeyCorner)) {
	case WhichKeyTopLeft, WhichKeyTopRight, WhichKeyBottomLeft, WhichKeyBottomRight:
		return strings.ToLower(strings.TrimSpace(l.WhichKeyCorner))
	default:
		return WhichKeyBottomRight
	}
}

// ValidWhichKeyCorner reports whether s is one of the four screen corners.
func ValidWhichKeyCorner(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case WhichKeyTopLeft, WhichKeyTopRight, WhichKeyBottomLeft, WhichKeyBottomRight:
		return true
	default:
		return false
	}
}
