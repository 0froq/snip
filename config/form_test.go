package config

import (
	"path/filepath"
	"testing"
)

func TestFormRoundTrip(t *testing.T) {
	src := Default()
	src.Card.Placement = "near-cursor"
	src.Secrets.ExtraPatterns = []string{"foo", "bar"}
	got, err := FormFrom(src).Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if got.EffectivePlacement() != PlacementNearCursor {
		t.Fatalf("placement %q", got.Card.Placement)
	}
	if len(got.Secrets.ExtraPatterns) != 2 || got.Secrets.ExtraPatterns[0] != "foo" {
		t.Fatalf("patterns %#v", got.Secrets.ExtraPatterns)
	}
	if got.Bindings.Settings.Direct != src.Bindings.Settings.Direct {
		t.Fatalf("settings shortcut %q", got.Bindings.Settings.Direct)
	}
}

func TestFormRejectsBadPlacement(t *testing.T) {
	f := FormFrom(Default())
	f.Placement = "beside"
	if _, err := f.Parse(Default()); err == nil {
		t.Fatal("expected error")
	}
}

func TestNeedsRestart(t *testing.T) {
	a := Default()
	b := a
	if NeedsRestart(a, b) {
		t.Fatal("same config")
	}
	off := !b.Window.EffectivePanel()
	b.Window.Panel = &off
	if !NeedsRestart(a, b) {
		t.Fatal("panel change")
	}
}

func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snip", "config.toml")
	src := Default()
	src.MaxCards = 7
	src.Window.AlwaysOnTop = false
	if err := Save(path, src); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.MaxCards != 7 || got.Window.AlwaysOnTop {
		t.Fatalf("reloaded %+v", got)
	}
}
