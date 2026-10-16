package config

import (
	"testing"

	"github.com/0froq/snip/internal/macpaste"
)

func TestPasteboardSkipTransient(t *testing.T) {
	p := DefaultPasteboard()
	if !p.SkipClipboardChange(macpaste.Markers{Transient: true}) {
		t.Fatal("expected skip")
	}
	if p.SkipClipboardChange(macpaste.Markers{Concealed: true}) {
		t.Fatal("concealed alone should not skip")
	}
	p.SkipTransient = false
	if p.SkipClipboardChange(macpaste.Markers{Transient: true}) {
		t.Fatal("disabled skip")
	}
}

func TestPasteboardConcealedOnCard(t *testing.T) {
	p := DefaultPasteboard()
	if !p.ConcealedOnCard(macpaste.Markers{Concealed: true}) {
		t.Fatal("expected concealed")
	}
	p.HonorConcealed = false
	if p.ConcealedOnCard(macpaste.Markers{Concealed: true}) {
		t.Fatal("honor off")
	}
}
