package config

import (
	"fmt"
	"strings"
)

// Palette ids. Auto follows the system appearance.
const (
	PaletteAuto        = "auto"
	PaletteSpringDay   = "spring-day"
	PaletteSpringNight = "spring-night"
	PaletteSummerDay   = "summer-day"
	PaletteSummerNight = "summer-night"
	PaletteAutumnDay   = "autumn-day"
	PaletteAutumnNight = "autumn-night"
	PaletteWinterDay   = "winter-day"
	PaletteWinterNight = "winter-night"
)

// PaletteIDs is the settings order, auto first.
var PaletteIDs = []string{
	PaletteAuto,
	PaletteSpringDay, PaletteSpringNight,
	PaletteSummerDay, PaletteSummerNight,
	PaletteAutumnDay, PaletteAutumnNight,
	PaletteWinterDay, PaletteWinterNight,
}

// Theme is the pixel palette selection.
type Theme struct {
	Palette string `toml:"palette"`
}

// EffectivePalette is a known id. An empty or unknown value is auto.
func (t Theme) EffectivePalette() string {
	id, err := NormalizePalette(t.Palette)
	if err != nil {
		return PaletteAuto
	}
	return id
}

// NormalizePalette accepts a palette id. Empty means auto.
func NormalizePalette(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		s = PaletteAuto
	}
	switch s {
	case PaletteAuto, PaletteSpringDay, PaletteSpringNight,
		PaletteSummerDay, PaletteSummerNight,
		PaletteAutumnDay, PaletteAutumnNight,
		PaletteWinterDay, PaletteWinterNight:
		return s, nil
	default:
		return "", fmt.Errorf("palette must be auto or a season")
	}
}
