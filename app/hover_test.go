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
	start := time.Unix(2, 0)
	wait := 2 * time.Second
	s := HoverState{Hidden: true}
	next, effect := stepHover(s, HoverSample{
		Now: start, Outside: true, Delay: time.Second, RestoreDelay: wait,
		Enabled: true, CardsShown: true,
	})
	if effect.Restore || effect.Hide || !next.Hidden || !next.OutsideSince.Equal(start) {
		t.Fatalf("leave starts the outside wait = %+v %+v", next, effect)
	}
	next, effect = stepHover(next, HoverSample{
		Now: start.Add(wait - time.Millisecond), Outside: true,
		Delay: time.Second, RestoreDelay: wait,
		Enabled: true, CardsShown: true,
	})
	if effect.Restore || !next.Hidden || !next.OutsideSince.Equal(start) {
		t.Fatalf("before restore delay = %+v %+v", next, effect)
	}
	next, effect = stepHover(next, HoverSample{
		Now: start.Add(wait), Outside: true,
		Delay: time.Second, RestoreDelay: wait,
		Enabled: true, CardsShown: true,
	})
	if !effect.Restore || effect.Hide || next.Hidden {
		t.Fatalf("restore delay elapsed = %+v %+v", next, effect)
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

func TestHoverButtonHeldDoesNotRestore(t *testing.T) {
	start := time.Unix(3, 0)
	wait := 2 * time.Second
	held := HoverSample{
		Now: start.Add(5 * time.Second), Outside: true,
		Delay: time.Second, RestoreDelay: wait,
		ButtonDown: true, Enabled: true, CardsShown: true,
	}
	s, effect := stepHover(HoverState{Hidden: true, OutsideSince: start}, held)
	if effect.Restore || effect.Hide || !s.Hidden || !s.OutsideSince.IsZero() {
		t.Fatalf("button held outside = %+v %+v", s, effect)
	}
	// Release while still outside. The wait starts at this sample, not at
	// the moment the pointer first left.
	released := held
	released.ButtonDown = false
	released.Now = start.Add(6 * time.Second)
	s, effect = stepHover(s, released)
	if effect.Restore || !s.Hidden || !s.OutsideSince.Equal(released.Now) {
		t.Fatalf("release starts the outside wait = %+v %+v", s, effect)
	}
	s, effect = stepHover(s, HoverSample{
		Now: released.Now.Add(wait), Outside: true,
		Delay: time.Second, RestoreDelay: wait,
		Enabled: true, CardsShown: true,
	})
	if !effect.Restore || s.Hidden {
		t.Fatalf("after release delay = %+v %+v", s, effect)
	}
}

func TestHoverOutsideTimerResetsOnReentry(t *testing.T) {
	start := time.Unix(4, 0)
	wait := 2 * time.Second
	s, effect := stepHover(HoverState{Hidden: true}, HoverSample{
		Now: start, Outside: true, RestoreDelay: wait,
		Enabled: true, CardsShown: true,
	})
	if effect.Restore || !s.Hidden || !s.OutsideSince.Equal(start) {
		t.Fatalf("outside = %+v %+v", s, effect)
	}
	// Back inside the expanded frame, including the margin. Stay hidden
	// and do not ask for another dwell.
	s, effect = stepHover(s, HoverSample{
		Now: start.Add(time.Second), RestoreDelay: wait,
		Enabled: true, CardsShown: true,
	})
	if effect.Restore || effect.Hide || !s.Hidden || !s.OutsideSince.IsZero() {
		t.Fatalf("re-enter = %+v %+v", s, effect)
	}
	left := start.Add(1500 * time.Millisecond)
	s, effect = stepHover(s, HoverSample{
		Now: left, Outside: true, RestoreDelay: wait,
		Enabled: true, CardsShown: true,
	})
	if effect.Restore || !s.OutsideSince.Equal(left) {
		t.Fatalf("leave again = %+v %+v", s, effect)
	}
	s, effect = stepHover(s, HoverSample{
		Now: left.Add(wait - time.Millisecond), Outside: true, RestoreDelay: wait,
		Enabled: true, CardsShown: true,
	})
	if effect.Restore || !s.Hidden {
		t.Fatalf("short second wait = %+v %+v", s, effect)
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

func TestHoverHideMargin(t *testing.T) {
	// Card at (100, 200), size 50x40. Margin 40 grows it to (60, 160, 130x120).
	const x, y, w, h, margin = 100, 200, 50, 40, 40
	if !pointInFrame(100, 200, x, y, w, h) || !pointInExpandedFrame(100, 200, x, y, w, h, margin) {
		t.Fatal("card origin")
	}
	// In the band around the card: outside the card, inside the restore frame.
	if pointInFrame(70, 200, x, y, w, h) || !pointInExpandedFrame(70, 200, x, y, w, h, margin) {
		t.Fatal("left margin")
	}
	if pointInFrame(100, 170, x, y, w, h) || !pointInExpandedFrame(100, 170, x, y, w, h, margin) {
		t.Fatal("top margin")
	}
	// The expanded right and bottom edges are outside. 60+130=190, 160+120=280.
	if pointInExpandedFrame(190, 200, x, y, w, h, margin) || pointInExpandedFrame(100, 280, x, y, w, h, margin) {
		t.Fatal("expanded edges")
	}
	if pointInExpandedFrame(59, 200, x, y, w, h, margin) {
		t.Fatal("past the margin")
	}
	if !pointInExpandedFrame(100, 200, x, y, w, h, 0) || pointInExpandedFrame(99, 200, x, y, w, h, 0) {
		t.Fatal("zero margin matches the card")
	}
}

func TestOpacityUpdateSkipsHoverAndHideAll(t *testing.T) {
	if !opacityUpdateAllowed(false, false, false, false, true) {
		t.Fatal("shown card")
	}
	if opacityUpdateAllowed(true, false, false, false, true) ||
		opacityUpdateAllowed(false, true, false, false, true) ||
		opacityUpdateAllowed(false, false, true, false, true) ||
		opacityUpdateAllowed(false, false, false, true, true) ||
		opacityUpdateAllowed(false, false, false, false, false) {
		t.Fatal("hide owns the alpha")
	}
}

func TestApplyHoverFadeDoesNotOrderOrFocus(t *testing.T) {
	w := &recordingHoverWindow{}
	applyHoverFade(w, HoverEffect{Hide: true}, 0.4, hoverFadeDuration)
	if !reflect.DeepEqual(w.calls, []string{"ignore true", "fade 0 200ms"}) {
		t.Fatalf("hide calls %v", w.calls)
	}
	w.calls = nil
	applyHoverFade(w, HoverEffect{Restore: true}, 0.4, hoverFadeDuration)
	if !reflect.DeepEqual(w.calls, []string{"ignore false", "fade 0.4 200ms"}) {
		t.Fatalf("restore calls %v", w.calls)
	}
	w.calls = nil
	applyHoverFade(w, HoverEffect{}, 1, hoverFadeDuration)
	if len(w.calls) != 0 {
		t.Fatalf("no effect wrote %v", w.calls)
	}
}

func TestHoverFadeCancelSupersedes(t *testing.T) {
	var h cardUI
	first := h.beginHoverFade()
	if !h.hoverFading || first == 0 {
		t.Fatalf("fade gen %d fading %v", first, h.hoverFading)
	}
	second := h.beginHoverFade()
	h.finishHoverFade(first)
	if !h.hoverFading || h.hoverFadeGen != second {
		t.Fatalf("older fade finished the newer one: %+v", h)
	}
	h.finishHoverFade(second)
	if h.hoverFading {
		t.Fatal("latest fade should finish")
	}
	h.beginHoverFade()
	h.cancelHoverFade()
	if h.hoverFading {
		t.Fatal("cancel should drop the fade")
	}
	h.finishHoverFade(second)
	if h.hoverFading {
		t.Fatal("a cancelled fade must stay idle")
	}
}

func TestHoverFadeDuration(t *testing.T) {
	if hoverFadeDuration != 200*time.Millisecond {
		t.Fatal(hoverFadeDuration)
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

func (w *recordingHoverWindow) FadeOpacity(alpha float64, d time.Duration) {
	w.calls = append(w.calls, "fade "+formatFade(alpha, d))
}

func formatFade(alpha float64, d time.Duration) string {
	if alpha == 0 && d == 200*time.Millisecond {
		return "0 200ms"
	}
	if alpha == 0.4 && d == 200*time.Millisecond {
		return "0.4 200ms"
	}
	return "other"
}

func (w *recordingHoverWindow) SetIgnoreMouseEvents(v bool) {
	if v {
		w.calls = append(w.calls, "ignore true")
		return
	}
	w.calls = append(w.calls, "ignore false")
}
