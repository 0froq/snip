package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEffectiveFramelessDefault(t *testing.T) {
	w := Window{}
	if !w.EffectiveFrameless() {
		t.Fatal("nil frameless should default true")
	}
	f := false
	explicit := Window{Frameless: &f}
	if explicit.EffectiveFrameless() {
		t.Fatal("explicit false")
	}
}

func TestFormSavePreservesFrameless(t *testing.T) {
	prev := Default()
	prev.MaxCards = 12
	f := FormFrom(prev)
	f.MaxCards = "15"
	got, err := f.Parse(prev)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Window.EffectiveFrameless() {
		t.Fatal("frameless should stay true")
	}
	if got.MaxCards != 15 {
		t.Fatalf("max cards %d", got.MaxCards)
	}
}

func TestLoadMissingFramelessUsesDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("max_cards = 5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Window.EffectiveFrameless() {
		t.Fatal("expected frameless default")
	}
	if got.MaxCards != 5 {
		t.Fatalf("max_cards %d", got.MaxCards)
	}
}

func TestNormalizeForSaveOmitsDefaultFrameless(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	cfg := Default()
	cfg.MaxCards = 3
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "frameless") {
		t.Fatalf("should omit default frameless: %s", body)
	}
}
