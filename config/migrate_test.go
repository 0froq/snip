package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadKeepsDeprecatedMaxLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	body := "[card]\nmax_lines = 7\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Card.MaxLines != 7 {
		t.Fatalf("max_lines %d", cfg.Card.MaxLines)
	}
	if cfg.Card.MaxHeight != Default().Card.MaxHeight {
		t.Fatalf("max_height %d", cfg.Card.MaxHeight)
	}
	text := strings.Repeat("word ", 80)
	layout := cfg.Card.Layout(text, 200, cfg.Card.MaxLines)
	if layout.Truncated || strings.Contains(layout.Text, "…") || layout.Text != text {
		t.Fatalf("old max_lines still clips: %+v", layout)
	}
	if layout.Height > cfg.Card.EffectiveMaxHeight() {
		t.Fatalf("height %d over max %d", layout.Height, cfg.Card.EffectiveMaxHeight())
	}
}

func TestLoadMaxHeightCapsTheCard(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("[card]\nmax_height = 80\nmax_lines = 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Card.MaxHeight != 80 || cfg.Card.MaxLines != 2 {
		t.Fatalf("card %+v", cfg.Card)
	}
	text := strings.Repeat("tall card ", 40)
	layout := cfg.Card.Layout(text, 200, 0)
	if layout.Height != 80 || !layout.Scrolls || layout.Text != text {
		t.Fatalf("layout %+v", layout)
	}
}

func TestParseKeepsMaxLinesWhenSavingHeight(t *testing.T) {
	prev := Default()
	prev.Card.MaxLines = 9
	f := FormFrom(prev)
	f.MaxHeight = "180"
	next, err := f.Parse(prev)
	if err != nil {
		t.Fatal(err)
	}
	if next.Card.MaxHeight != 180 || next.Card.MaxLines != 9 {
		t.Fatalf("card %+v", next.Card)
	}
}
