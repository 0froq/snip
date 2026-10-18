package config

import "testing"

func TestNormalizePalette(t *testing.T) {
	id, err := NormalizePalette("  Autumn-Day ")
	if err != nil || id != PaletteAutumnDay {
		t.Fatalf("id %q err %v", id, err)
	}
	if _, err := NormalizePalette(""); err != nil {
		t.Fatal(err)
	}
	if got := (Theme{}).EffectivePalette(); got != PaletteAuto {
		t.Fatal(got)
	}
	if got := (Theme{Palette: "nope"}).EffectivePalette(); got != PaletteAuto {
		t.Fatal(got)
	}
	if _, err := NormalizePalette("nope"); err == nil {
		t.Fatal("unknown palette accepted")
	}
	if len(PaletteIDs) != 9 {
		t.Fatal(len(PaletteIDs))
	}
}
