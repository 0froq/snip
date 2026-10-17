package app

import (
	"reflect"
	"testing"
)

type recordingNoteWindow struct {
	calls []string
}

func (w *recordingNoteWindow) ShowInactive() { w.calls = append(w.calls, "show inactive") }
func (w *recordingNoteWindow) Invalidate()   { w.calls = append(w.calls, "invalidate") }
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
		want := []string{"show inactive", "invalidate", "native display"}
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
