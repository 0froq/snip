package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOpacityDefaultsAndClamp(t *testing.T) {
	var blank Card
	if blank.EffectiveActiveOpacity() != 1 || blank.EffectiveInactiveOpacity() != 1 {
		t.Fatalf("unset active %v inactive %v", blank.EffectiveActiveOpacity(), blank.EffectiveInactiveOpacity())
	}
	if blank.OpacityFor(true) != 1 || blank.OpacityFor(false) != 1 {
		t.Fatal("unset selection opacity")
	}
	low := Card{ActiveOpacity: 0.05, InactiveOpacity: 2}
	if low.EffectiveActiveOpacity() != OpacityMin || low.EffectiveInactiveOpacity() != OpacityMax {
		t.Fatalf("clamp active %v inactive %v", low.EffectiveActiveOpacity(), low.EffectiveInactiveOpacity())
	}
	set := Card{ActiveOpacity: 0.8, InactiveOpacity: 0.3}
	if set.OpacityFor(true) != 0.8 || set.OpacityFor(false) != 0.3 {
		t.Fatalf("selected %v idle %v", set.OpacityFor(true), set.OpacityFor(false))
	}
}

func TestOpacityFormRoundTrip(t *testing.T) {
	src := Default()
	src.Card.ActiveOpacity = 0.85
	src.Card.InactiveOpacity = 0.4
	got, err := FormFrom(src).Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if got.Card.ActiveOpacity != 0.85 || got.Card.InactiveOpacity != 0.4 {
		t.Fatalf("round trip %+v", got.Card)
	}
	f := FormFrom(src)
	f.ActiveOpacity = "0.1"
	if _, err := f.Parse(src); err == nil {
		t.Fatal("opacity below 0.2")
	}
	f.ActiveOpacity = "1"
	f.InactiveOpacity = "1.2"
	if _, err := f.Parse(src); err == nil {
		t.Fatal("opacity above 1")
	}
	f.InactiveOpacity = "1"
	got, err = f.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if got.Card.OpacityFor(true) != 1 || got.Card.OpacityFor(false) != 1 {
		t.Fatalf("edge %+v", got.Card)
	}
}

func TestHoverHideDefaults(t *testing.T) {
	if !Default().Card.HoverHide || Default().Card.EffectiveHoverHideDelayMs() != DefaultHoverHideDelayMs {
		t.Fatalf("default %+v", Default().Card)
	}
	var blank Card
	if blank.HoverHide {
		t.Fatal("zero value is not the file default; Load starts from Default")
	}
	if blank.HoverHideDelay() != 1500*time.Millisecond {
		t.Fatal(blank.HoverHideDelay())
	}
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("max_cards = 3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Card.HoverHide || got.Card.HoverHideDelayMs != DefaultHoverHideDelayMs {
		t.Fatalf("missing keys %+v", got.Card)
	}
	off := Default()
	off.Card.HoverHide = false
	off.Card.HoverHideDelayMs = 800
	round, err := FormFrom(off).Parse(off)
	if err != nil {
		t.Fatal(err)
	}
	if round.Card.HoverHide || round.Card.HoverHideDelayMs != 800 {
		t.Fatalf("round trip %+v", round.Card)
	}
	f := FormFrom(off)
	f.HoverHideDelayMs = "0"
	if _, err := f.Parse(off); err == nil {
		t.Fatal("delay must be greater than zero")
	}
}

func TestOpacityMissingKeyStaysOpaque(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("max_cards = 3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Card.ActiveOpacity != 1 || got.Card.InactiveOpacity != 1 {
		t.Fatalf("missing keys %+v", got.Card)
	}
	if got.Card.OpacityFor(true) != 1 || got.Card.OpacityFor(false) != 1 {
		t.Fatalf("loaded %+v", got.Card)
	}
}
