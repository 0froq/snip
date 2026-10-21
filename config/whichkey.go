package config

import "strings"

const (
	AlignStart  = "start"
	AlignCenter = "center"
	AlignEnd    = "end"
)

// WhichKeyPercents clamps the panel anchor to 0..100 percent of the work area.
func (l Leader) WhichKeyPercents() (x, y int) {
	return clampPercent(l.WhichKeyX), clampPercent(l.WhichKeyY)
}

// EffectiveAlignX is which horizontal point of the panel sits at WhichKeyX.
// Unrecognized values use end, the bottom-right default.
func (l Leader) EffectiveAlignX() string { return effectiveAlign(l.WhichKeyAlignX) }

// EffectiveAlignY is which vertical point of the panel sits at WhichKeyY.
func (l Leader) EffectiveAlignY() string { return effectiveAlign(l.WhichKeyAlignY) }

func clampPercent(n int) int {
	if n < 0 {
		return 0
	}
	if n > 100 {
		return 100
	}
	return n
}

func effectiveAlign(s string) string {
	if a, ok := NormalizeAlign(s); ok {
		return a
	}
	return AlignEnd
}

// NormalizeAlign accepts start, center, or end.
func NormalizeAlign(s string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case AlignStart, AlignCenter, AlignEnd:
		return strings.ToLower(strings.TrimSpace(s)), true
	default:
		return "", false
	}
}

// ApplyLegacyWhichKeyCorner maps the old four-corner option onto percent
// anchors. Unknown corners leave the leader unchanged.
func (l *Leader) ApplyLegacyWhichKeyCorner(corner string) {
	switch strings.ToLower(strings.TrimSpace(corner)) {
	case "top-left":
		l.WhichKeyX, l.WhichKeyY = 0, 0
		l.WhichKeyAlignX, l.WhichKeyAlignY = AlignStart, AlignStart
	case "top-right":
		l.WhichKeyX, l.WhichKeyY = 100, 0
		l.WhichKeyAlignX, l.WhichKeyAlignY = AlignEnd, AlignStart
	case "bottom-left":
		l.WhichKeyX, l.WhichKeyY = 0, 100
		l.WhichKeyAlignX, l.WhichKeyAlignY = AlignStart, AlignEnd
	case "bottom-right":
		l.WhichKeyX, l.WhichKeyY = 100, 100
		l.WhichKeyAlignX, l.WhichKeyAlignY = AlignEnd, AlignEnd
	}
}
