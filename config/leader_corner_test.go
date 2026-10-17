package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWhichKeyCornerDefault(t *testing.T) {
	got := Default().Leader.EffectiveWhichKeyCorner()
	if got != WhichKeyBottomRight {
		t.Fatalf("default corner %q", got)
	}
	if Default().Leader.WhichKeyCorner != WhichKeyBottomRight {
		t.Fatalf("default field %q", Default().Leader.WhichKeyCorner)
	}
}

func TestWhichKeyCornerParse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	body := "[leader]\nwhich_key_corner = \"top-left\"\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Leader.EffectiveWhichKeyCorner() != WhichKeyTopLeft {
		t.Fatalf("parsed %q", got.Leader.WhichKeyCorner)
	}

	if err := os.WriteFile(path, []byte("[leader]\ntimeout_ms = 900\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Leader.EffectiveWhichKeyCorner() != WhichKeyBottomRight {
		t.Fatalf("omitted corner %q", got.Leader.WhichKeyCorner)
	}
	if got.Leader.TimeoutMs != 900 {
		t.Fatalf("timeout %d", got.Leader.TimeoutMs)
	}

	if err := os.WriteFile(path, []byte("[leader]\nwhich_key_corner = \"CENTER\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Leader.EffectiveWhichKeyCorner() != WhichKeyBottomRight {
		t.Fatalf("invalid corner %q", got.Leader.EffectiveWhichKeyCorner())
	}
}

func TestWhichKeyCornerForm(t *testing.T) {
	f := FormFrom(Default())
	if f.WhichKeyCorner != WhichKeyBottomRight {
		t.Fatalf("form default %q", f.WhichKeyCorner)
	}
	f.WhichKeyCorner = "Top-Left"
	got, err := f.Parse(Default())
	if err != nil {
		t.Fatal(err)
	}
	if got.Leader.WhichKeyCorner != WhichKeyTopLeft {
		t.Fatalf("parsed form %q", got.Leader.WhichKeyCorner)
	}
	f.WhichKeyCorner = "middle"
	if _, err := f.Parse(Default()); err == nil {
		t.Fatal("expected error")
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
	if f.FontFamily != DefaultFontFamily || f.FontSize != "14" {
		t.Fatalf("form %q %q", f.FontFamily, f.FontSize)
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

func TestRepoConfigWhichKeyCorner(t *testing.T) {
	got, err := Load(filepath.Join("..", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Leader.EffectiveWhichKeyCorner() != WhichKeyBottomRight {
		t.Fatalf("repo config %q", got.Leader.WhichKeyCorner)
	}
	if got.Card.FontSize != DefaultFontSize || got.Card.EffectiveFontFamily() != DefaultFontFamily {
		t.Fatalf("repo font %q %d", got.Card.FontFamily, got.Card.FontSize)
	}
}
