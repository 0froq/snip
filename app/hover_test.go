package app

import (
	"reflect"
	"testing"
	"time"
)

func TestHoverDwellCountsWhileThePointerStaysInside(t *testing.T) {
	start := time.Unix(0, 0)
	delay := 1500 * time.Millisecond
	sample := func(at time.Duration, inside bool) HoverSample {
		return HoverSample{
			Now: start.Add(at), Inside: inside, Delay: delay,
			Enabled: true, CardsShown: true,
		}
	}

	s, effect := stepHover(HoverState{}, sample(0, true))
	if effect.Hide || effect.Restore || s.Hidden || !s.InsideSince.Equal(start) {
		t.Fatalf("enter = %+v %+v", s, effect)
	}
	// A later sample is still inside. The clock does not restart just
	// because the pointer moved.
	s, effect = stepHover(s, sample(400*time.Millisecond, true))
	if effect.Hide || s.Hidden || !s.InsideSince.Equal(start) {
		t.Fatalf("move inside = %+v %+v", s, effect)
	}
	s, effect = stepHover(s, sample(1499*time.Millisecond, true))
	if effect.Hide || s.Hidden {
		t.Fatalf("before delay = %+v %+v", s, effect)
	}
	s, effect = stepHover(s, sample(1500*time.Millisecond, true))
	if !effect.Hide || !s.Hidden || effect.Restore {
		t.Fatalf("at delay = %+v %+v", s, effect)
	}
	s, effect = stepHover(s, sample(2*time.Second, true))
	if effect.Hide || effect.Restore || !s.Hidden {
		t.Fatalf("stay hidden = %+v %+v", s, effect)
	}
}

func TestHoverLeaveRestores(t *testing.T) {
	s := HoverState{Hidden: true, InsideSince: time.Unix(0, 0)}
	next, effect := stepHover(s, HoverSample{
		Now: time.Unix(2, 0), Delay: time.Second,
		Enabled: true, CardsShown: true,
	})
	if !effect.Restore || effect.Hide || next.Hidden || !next.InsideSince.IsZero() {
		t.Fatalf("leave = %+v %+v", next, effect)
	}
}

func TestHoverBlockedDoesNotStart(t *testing.T) {
	start := time.Unix(10, 0)
	s, effect := stepHover(HoverState{}, HoverSample{
		Now: start, Inside: true, Delay: time.Second,
		Blocked: true, Enabled: true, CardsShown: true,
	})
	if effect.Hide || effect.Restore || s.Hidden || !s.InsideSince.IsZero() {
		t.Fatalf("blocked enter = %+v %+v", s, effect)
	}
	counting := HoverState{InsideSince: start}
	s, effect = stepHover(counting, HoverSample{
		Now: start.Add(2 * time.Second), Inside: true, Delay: time.Second,
		Blocked: true, Enabled: true, CardsShown: true,
	})
	if effect.Hide || s.Hidden || !s.InsideSince.IsZero() {
		t.Fatalf("blocked during dwell = %+v %+v", s, effect)
	}
}

func TestHoverBlockedLeaveStillRestores(t *testing.T) {
	s, effect := stepHover(HoverState{Hidden: true}, HoverSample{
		Now: time.Unix(3, 0), Delay: time.Second,
		Blocked: true, Enabled: true, CardsShown: true,
	})
	if !effect.Restore || s.Hidden {
		t.Fatalf("blocked leave = %+v %+v", s, effect)
	}
	s, effect = stepHover(HoverState{Hidden: true}, HoverSample{
		Now: time.Unix(3, 0), Inside: true, Delay: time.Second,
		Blocked: true, Enabled: true, CardsShown: true,
	})
	if effect.Restore || effect.Hide || !s.Hidden {
		t.Fatalf("blocked while still inside = %+v %+v", s, effect)
	}
}

func TestHoverHideAllDoesNotRestore(t *testing.T) {
	s, effect := stepHover(HoverState{Hidden: true, InsideSince: time.Unix(1, 0)}, HoverSample{
		Now: time.Unix(4, 0), Inside: false, Delay: time.Second,
		Enabled: true, CardsShown: false,
	})
	if effect.Restore || effect.Hide || s.Hidden || !s.InsideSince.IsZero() {
		t.Fatalf("hide-all = %+v %+v", s, effect)
	}
}

func TestHoverDisabledRestores(t *testing.T) {
	s, effect := stepHover(HoverState{Hidden: true}, HoverSample{
		Now: time.Unix(1, 0), Inside: true, Delay: time.Second,
		Enabled: false, CardsShown: true,
	})
	if !effect.Restore || s.Hidden {
		t.Fatalf("disabled = %+v %+v", s, effect)
	}
}

func TestPointerBlocksHover(t *testing.T) {
	if pointerBlocksHover(false, false, false, false, false) {
		t.Fatal("idle pointer")
	}
	blocks := []struct{ drag, resize, edit, menu, button bool }{
		{drag: true},
		{resize: true},
		{edit: true},
		{menu: true},
		{button: true},
	}
	for _, b := range blocks {
		if !pointerBlocksHover(b.drag, b.resize, b.edit, b.menu, b.button) {
			t.Fatalf("expected block %+v", b)
		}
	}
	host := &cardUI{pointerActive: true}
	if !host.hoverBlock(false) {
		t.Fatal("drag")
	}
	host = &cardUI{resizeDepth: 1}
	if !host.hoverBlock(false) {
		t.Fatal("resize")
	}
	host = &cardUI{pixelEdit: true}
	if !host.hoverBlock(false) {
		t.Fatal("edit")
	}
	host = &cardUI{menuOpen: true}
	if !host.hoverBlock(false) {
		t.Fatal("menu")
	}
	host = &cardUI{}
	if !host.hoverBlock(true) {
		t.Fatal("button")
	}
	if host.hoverBlock(false) {
		t.Fatal("clear")
	}
}

func TestPointInFrame(t *testing.T) {
	if !pointInFrame(10, 20, 10, 20, 30, 40) {
		t.Fatal("origin is inside")
	}
	if pointInFrame(40, 20, 10, 20, 30, 40) || pointInFrame(10, 60, 10, 20, 30, 40) {
		t.Fatal("right and bottom edges are outside")
	}
	if pointInFrame(9, 20, 10, 20, 30, 40) || pointInFrame(0, 0, 0, 0, 0, 10) {
		t.Fatal("outside or empty")
	}
}

func TestOpacityUpdateSkipsHoverAndHideAll(t *testing.T) {
	if !opacityUpdateAllowed(false, false, false, true) {
		t.Fatal("shown card")
	}
	if opacityUpdateAllowed(true, false, false, true) ||
		opacityUpdateAllowed(false, true, false, true) ||
		opacityUpdateAllowed(false, false, true, true) ||
		opacityUpdateAllowed(false, false, false, false) {
		t.Fatal("hide owns the alpha")
	}
}

func TestApplyHoverEffectDoesNotOrderOrFocus(t *testing.T) {
	w := &recordingHoverWindow{}
	applyHoverEffect(w, HoverEffect{Hide: true}, 0.4)
	if !reflect.DeepEqual(w.calls, []string{"ignore true", "opacity 0"}) {
		t.Fatalf("hide calls %v", w.calls)
	}
	w.calls = nil
	applyHoverEffect(w, HoverEffect{Restore: true}, 0.4)
	if !reflect.DeepEqual(w.calls, []string{"ignore false", "opacity 0.4"}) {
		t.Fatalf("restore calls %v", w.calls)
	}
	w.calls = nil
	applyHoverEffect(w, HoverEffect{}, 1)
	if len(w.calls) != 0 {
		t.Fatalf("no effect wrote %v", w.calls)
	}
}

func TestHoverPollInterval(t *testing.T) {
	if hoverPollInterval < 50*time.Millisecond || hoverPollInterval > 100*time.Millisecond {
		t.Fatal(hoverPollInterval)
	}
}

type recordingHoverWindow struct {
	calls []string
}

func (w *recordingHoverWindow) SetOpacity(v float64) {
	if v == 0 {
		w.calls = append(w.calls, "opacity 0")
		return
	}
	w.calls = append(w.calls, "opacity 0.4")
}

func (w *recordingHoverWindow) SetIgnoreMouseEvents(v bool) {
	if v {
		w.calls = append(w.calls, "ignore true")
		return
	}
	w.calls = append(w.calls, "ignore false")
}
