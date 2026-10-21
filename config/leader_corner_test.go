package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWhichKeyDefault(t *testing.T) {
	l := Default().Leader
	x, y := l.WhichKeyPercents()
	if x != 100 || y != 100 || l.EffectiveAlignX() != AlignEnd || l.EffectiveAlignY() != AlignEnd {
		t.Fatalf("default placement %d,%d %s %s", x, y, l.EffectiveAlignX(), l.EffectiveAlignY())
	}
}

func TestWhichKeyParseAndLegacyCorner(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	body := "[leader]\nwhich_key_x = 10\nwhich_key_y = 40\nwhich_key_align_x = \"center\"\nwhich_key_align_y = \"start\"\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	x, y := got.Leader.WhichKeyPercents()
	if x != 10 || y != 40 || got.Leader.EffectiveAlignX() != AlignCenter || got.Leader.EffectiveAlignY() != AlignStart {
		t.Fatalf("parsed %d,%d %s %s", x, y, got.Leader.EffectiveAlignX(), got.Leader.EffectiveAlignY())
	}

	// A corner alone still maps onto the percent anchor.
	if err := os.WriteFile(path, []byte("[leader]\nwhich_key_corner = \"top-left\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = Load(path)
	if err != nil {
		t.Fatal(err)
	}
	x, y = got.Leader.WhichKeyPercents()
	if x != 0 || y != 0 || got.Leader.EffectiveAlignX() != AlignStart || got.Leader.EffectiveAlignY() != AlignStart {
		t.Fatalf("legacy corner %d,%d %s %s", x, y, got.Leader.EffectiveAlignX(), got.Leader.EffectiveAlignY())
	}

	// Explicit percent keys win over a leftover corner.
	if err := os.WriteFile(path, []byte("[leader]\nwhich_key_corner = \"top-left\"\nwhich_key_x = 100\nwhich_key_y = 100\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = Load(path)
	if err != nil {
		t.Fatal(err)
	}
	x, y = got.Leader.WhichKeyPercents()
	if x != 100 || y != 100 || got.Leader.EffectiveAlignX() != AlignEnd {
		t.Fatalf("percent overrides corner: %d,%d %s", x, y, got.Leader.EffectiveAlignX())
	}
}

func TestWhichKeyForm(t *testing.T) {
	f := FormFrom(Default())
	if f.WhichKeyX != "100" || f.WhichKeyY != "100" || f.WhichKeyAlignX != AlignEnd || f.WhichKeyAlignY != AlignEnd {
		t.Fatalf("form default %+v", f)
	}
	f.WhichKeyX = "0"
	f.WhichKeyY = "50"
	f.WhichKeyAlignX = "Start"
	f.WhichKeyAlignY = "center"
	got, err := f.Parse(Default())
	if err != nil {
		t.Fatal(err)
	}
	if got.Leader.WhichKeyX != 0 || got.Leader.WhichKeyY != 50 || got.Leader.WhichKeyAlignX != AlignStart || got.Leader.WhichKeyAlignY != AlignCenter {
		t.Fatalf("parsed form %+v", got.Leader)
	}
	f.WhichKeyAlignX = "middle"
	if _, err := f.Parse(Default()); err == nil {
		t.Fatal("expected align error")
	}
	f.WhichKeyAlignX = AlignStart
	f.WhichKeyX = "120"
	if _, err := f.Parse(Default()); err == nil {
		t.Fatal("expected percent error")
	}
}

func TestFontFamilyDefaultAndForm(t *testing.T) {
	d := Default()
	if d.Card.FontSize != DefaultFontSize || d.Card.EffectiveFontFamily() != DefaultFontFamily {
		t.Fatalf("default font %q %d", d.Card.FontFamily, d.Card.FontSize)
	}
	if d.Card.EffectiveFontSize() != DefaultFontSize {
		t.Fatalf("size %d", d.Card.EffectiveFontSize())
	}
	blank := d.Card
	blank.FontFamily = "  "
	blank.FontSize = 0
	if blank.EffectiveFontFamily() != DefaultFontFamily || blank.EffectiveFontSize() != DefaultFontSize {
		t.Fatalf("fallback %q %d", blank.EffectiveFontFamily(), blank.EffectiveFontSize())
	}
	f := FormFrom(d)
	if d.Card.EffectiveCardFont() != CardFontSystem || d.Card.PixelFont() {
		t.Fatalf("default card font %q", d.Card.FontKind)
	}
	if f.FontFamily != DefaultFontFamily || f.FontSize != "14" || f.CardFont != CardFontSystem {
		t.Fatalf("form %q %q %q", f.FontFamily, f.FontSize, f.CardFont)
	}
	f.FontFamily = "  Helvetica Neue, PingFang SC  "
	f.FontSize = "16"
	got, err := f.Parse(d)
	if err != nil {
		t.Fatal(err)
	}
	if got.Card.FontFamily != "Helvetica Neue, PingFang SC" || got.Card.FontSize != 16 {
		t.Fatalf("parsed %q %d", got.Card.FontFamily, got.Card.FontSize)
	}
	if got.Card.UIFont(0).Family != got.Card.FontFamily || got.Card.UIFont(0).Size != 16 {
		t.Fatalf("ui font %+v", got.Card.UIFont(0))
	}
}

func TestRepoConfigWhichKey(t *testing.T) {
	got, err := Load(filepath.Join("..", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	x, y := got.Leader.WhichKeyPercents()
	if x != 100 || y != 100 || got.Leader.EffectiveAlignX() != AlignEnd || got.Leader.EffectiveAlignY() != AlignEnd {
		t.Fatalf("repo config %d,%d %s %s", x, y, got.Leader.EffectiveAlignX(), got.Leader.EffectiveAlignY())
	}
	if got.Card.FontSize != DefaultFontSize || got.Card.EffectiveFontFamily() != DefaultFontFamily {
		t.Fatalf("repo font %q %d", got.Card.FontFamily, got.Card.FontSize)
	}
}
