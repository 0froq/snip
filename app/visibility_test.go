package app

import (
	"reflect"
	"testing"

	"github.com/0froq/snip/config"
)

type recordingNoteWindow struct {
	calls []string
}

func (w *recordingNoteWindow) ShowInactive()      { w.calls = append(w.calls, "show inactive") }
func (w *recordingNoteWindow) Invalidate()        { w.calls = append(w.calls, "invalidate") }
func (w *recordingNoteWindow) SetOpacity(float64) { w.calls = append(w.calls, "opacity") }
func (w *recordingNoteWindow) NativeHandle() uintptr {
	w.calls = append(w.calls, "native display")
	return 0 // No AppKit/display server needed, including on Darwin.
}

func TestShowNoteWindowAlwaysRequestsFreshContent(t *testing.T) {
	w := &recordingNoteWindow{}
	h := &cardUI{hiding: true, hideGen: 40}
	// First show, show after hide, and resurfacing an already visible note
	// must all request a new UI frame, without ever using a focus-taking API.
	for range 3 {
		w.calls = nil
		gen := h.hideGen
		h.showWindow(w)
		if h.hiding || h.hideGen != gen+1 {
			t.Fatalf("show must reset hiding and supersede pending hides: hiding=%v generation=%d (was %d)", h.hiding, h.hideGen, gen)
		}
		want := []string{"invalidate", "native display", "show inactive", "opacity"}
		if !reflect.DeepEqual(w.calls, want) {
			t.Fatalf("show calls = %v, want %v", w.calls, want)
		}
	}
}

func TestShowCardIgnoresMissingWindow(t *testing.T) {
	var a App
	a.showCard(nil)
	a.showCard(&cardUI{})
}

func TestCopyWhileHiddenStaysOrderedOut(t *testing.T) {
	a := New(config.Config{}, "")
	if !a.cardsShown() {
		t.Fatal("cards start shown")
	}
	a.mu.Lock()
	a.visible = false
	a.cards["copied"] = &cardUI{}
	a.order = []string{"copied"}
	a.mu.Unlock()

	called := false
	a.orderCardFront(a.cards["copied"], func(noteWindow) { called = true })
	if called {
		t.Fatal("a copy while cards are hidden must stay ordered out")
	}
	if a.cardsShown() {
		t.Fatal("a copy must not turn hide-all off")
	}

	a.mu.Lock()
	a.visible = true
	ids := append([]string(nil), a.order...)
	a.mu.Unlock()
	if !a.cardsShown() {
		t.Fatal("show-all turns the global state on")
	}
	if len(ids) != 1 || ids[0] != "copied" {
		t.Fatalf("show-all must include the card copied while hidden, ids=%v", ids)
	}
	w := &recordingNoteWindow{}
	a.orderCardFront(a.cards["copied"], func(noteWindow) { a.cards["copied"].showWindow(w) })
	want := []string{"invalidate", "native display", "show inactive", "opacity"}
	if !reflect.DeepEqual(w.calls, want) {
		t.Fatalf("show-all present calls = %v, want %v", w.calls, want)
	}
}
