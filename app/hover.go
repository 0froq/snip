package app

import (
	"time"

	"github.com/0froq/snip/internal/macplace"
	"github.com/egoist/mygo"
)

const (
	// hoverPollInterval is how often the global cursor is sampled. A hidden
	// card receives no mouse events, and a pointer that rests still does not
	// generate moves, so the dwell has to be counted from this poll.
	hoverPollInterval = 80 * time.Millisecond
	// hoverFadeDuration is how long a hover hide or restore takes to fade.
	hoverFadeDuration = 200 * time.Millisecond
)

// HoverState is one card's dwell. InsideSince is when the pointer entered
// the card frame. It stays put while the pointer moves inside that frame.
// Hidden means the card has been dropped to alpha 0 and ignores the mouse.
// OutsideSince is when the pointer left the expanded frame while hidden.
type HoverState struct {
	InsideSince  time.Time
	OutsideSince time.Time
	Hidden       bool
}

// HoverSample is one poll of a card. Inside is the card frame, where the
// hide dwell counts. Outside is past the frame plus its margin, where the
// restore wait counts. Blocked is drag, resize, text editing, or an open
// context menu. ButtonDown is a held mouse button and, while the card is
// already hidden, forbids restore.
type HoverSample struct {
	Now          time.Time
	Inside       bool
	Outside      bool
	Delay        time.Duration
	RestoreDelay time.Duration
	Blocked      bool
	ButtonDown   bool
	Enabled      bool
	CardsShown   bool
}

// HoverEffect is what the poll should do to the window. Hide and Restore
// are never both set. Neither means the window is left alone, so a hide-all
// that already ordered the card out is not shown again.
type HoverEffect struct {
	Hide    bool
	Restore bool
}

// stepHover advances a card's dwell. Time inside the card frame counts
// whether or not the pointer is moving. A hidden card stays hidden until
// the pointer has remained outside the expanded frame for RestoreDelay,
// and a held mouse button pauses that wait. Re-entering the expanded frame
// clears it without asking for another dwell. Hide-all clears the dwell
// and does not restore.
func stepHover(s HoverState, in HoverSample) (HoverState, HoverEffect) {
	if !in.CardsShown {
		return HoverState{}, HoverEffect{}
	}
	if !in.Enabled {
		if s.Hidden {
			return HoverState{}, HoverEffect{Restore: true}
		}
		return HoverState{}, HoverEffect{}
	}
	outside := in.Outside && !in.Inside
	if s.Hidden {
		if in.ButtonDown || !outside {
			return HoverState{Hidden: true}, HoverEffect{}
		}
		since := s.OutsideSince
		if since.IsZero() {
			since = in.Now
		}
		wait := in.RestoreDelay
		if wait < 0 {
			wait = 0
		}
		if !in.Now.Before(since.Add(wait)) {
			return HoverState{}, HoverEffect{Restore: true}
		}
		return HoverState{Hidden: true, OutsideSince: since}, HoverEffect{}
	}
	if in.Blocked || in.ButtonDown || !in.Inside {
		return HoverState{}, HoverEffect{}
	}
	since := s.InsideSince
	if since.IsZero() {
		since = in.Now
	}
	if in.Delay > 0 && !in.Now.Before(since.Add(in.Delay)) {
		return HoverState{Hidden: true}, HoverEffect{Hide: true}
	}
	return HoverState{InsideSince: since}, HoverEffect{}
}

// pointerBlocksHover reports the conditions that must not start a hover hide.
func pointerBlocksHover(dragging, resizing, editing, menu, buttonDown bool) bool {
	return dragging || resizing || editing || menu || buttonDown
}

// pointInFrame reports whether a top-left screen point lies in a window frame.
// The right and bottom edges are outside.
func pointInFrame(px, py, x, y, w, h int) bool {
	if w <= 0 || h <= 0 {
		return false
	}
	return px >= x && py >= y && px < x+w && py < y+h
}

// pointInExpandedFrame reports whether a point lies in the frame grown by
// margin on every side. The margin is the restore buffer: the band around
// the card still counts as inside for coming back.
func pointInExpandedFrame(px, py, x, y, w, h, margin int) bool {
	if margin < 0 {
		margin = 0
	}
	return pointInFrame(px, py, x-margin, y-margin, w+2*margin, h+2*margin)
}

// opacityUpdateAllowed is false while a hide, hide-all, or hover fade owns
// the window alpha. A running fade must not be snapped by a selection change.
func opacityUpdateAllowed(hiding, orderedOut, hoverHidden, hoverFading, cardsShown bool) bool {
	return cardsShown && !hiding && !orderedOut && !hoverHidden && !hoverFading
}

// hoverFader fades alpha. It has no Show or Focus, so a restore cannot
// take keyboard focus.
type hoverFader interface {
	SetIgnoreMouseEvents(bool)
	FadeOpacity(alpha float64, d time.Duration)
}

// applyHoverFade hides by ignoring the mouse and fading alpha to 0, or
// restores mouse events and fades back to the configured alpha. The ignore
// flag changes at the start of the fade. A later call replaces an in-flight
// fade instead of waiting for it.
func applyHoverFade(win hoverFader, effect HoverEffect, opacity float64, d time.Duration) {
	if win == nil {
		return
	}
	if effect.Hide {
		win.SetIgnoreMouseEvents(true)
		win.FadeOpacity(0, d)
		return
	}
	if effect.Restore {
		win.SetIgnoreMouseEvents(false)
		win.FadeOpacity(opacity, d)
	}
}

type windowAlphaFader struct {
	win *mygo.Window
}

func (f windowAlphaFader) SetIgnoreMouseEvents(v bool) {
	if f.win != nil {
		f.win.SetIgnoreMouseEvents(v)
	}
}

func (f windowAlphaFader) FadeOpacity(alpha float64, d time.Duration) {
	if f.win != nil {
		macplace.AnimateWindowAlpha(f.win.NativeHandle(), alpha, d)
	}
}

func (h *cardUI) beginHoverFade() uint64 {
	h.hoverFadeGen++
	h.hoverFading = true
	return h.hoverFadeGen
}

// finishHoverFade marks the fade idle only when gen is still the latest.
// An older fade that a pointer flip replaced must not clear the new one.
func (h *cardUI) finishHoverFade(gen uint64) {
	if h == nil || h.hoverFadeGen != gen {
		return
	}
	h.hoverFading = false
}

// cancelHoverFade drops an in-flight hover fade so hide-all and show can
// set alpha themselves.
func (h *cardUI) cancelHoverFade() {
	if h == nil {
		return
	}
	h.hoverFadeGen++
	h.hoverFading = false
}

func (h *cardUI) hoverBlock(buttonDown bool) bool {
	if h == nil {
		return true
	}
	editing := h.pixelEdit || (h.textHost != nil && h.textHost.Editing())
	resizing := h.resizeDepth > 0
	if h.win != nil {
		resizing = resizing || macplace.InLiveResize(h.win.NativeHandle())
	}
	return pointerBlocksHover(h.pointerActive, resizing, editing, h.menuOpen, buttonDown)
}

func (a *App) ensureHoverPoll() {
	want := a.wantHoverPoll()
	if want && a.hoverStop == nil {
		stop := make(chan struct{})
		a.hoverStop = stop
		go a.hoverLoop(stop)
	}
	if !want && a.hoverStop != nil {
		close(a.hoverStop)
		a.hoverStop = nil
	}
}

func (a *App) wantHoverPoll() bool {
	if !a.config().Card.HoverHide || !a.cardsShown() {
		return false
	}
	a.mu.Lock()
	n := len(a.cards)
	a.mu.Unlock()
	return n > 0
}

func (a *App) hoverLoop(stop <-chan struct{}) {
	tick := time.NewTicker(hoverPollInterval)
	defer tick.Stop()
	for {
		select {
		case <-stop:
			return
		case <-tick.C:
			mygo.RunOnMain(a.pollHover)
		}
	}
}

func (a *App) pollHover() {
	if !a.wantHoverPoll() {
		a.ensureHoverPoll()
		return
	}
	now := time.Now()
	pt := mygo.Screen.CursorScreenPoint()
	buttons := macplace.MouseButtonsDown()
	cfg := a.config().Card
	a.mu.Lock()
	ids := append([]string(nil), a.order...)
	a.mu.Unlock()
	shown := a.cardsShown()
	for _, id := range ids {
		a.mu.Lock()
		h := a.cards[id]
		a.mu.Unlock()
		if h == nil || h.win == nil || h.orderedOut || h.hiding {
			if h != nil {
				h.hover = HoverState{}
			}
			continue
		}
		x, y := h.win.Position()
		w, height := h.win.Size()
		margin := cfg.EffectiveHoverHideMargin()
		inside := pointInFrame(pt.X, pt.Y, x, y, w, height)
		outside := !pointInExpandedFrame(pt.X, pt.Y, x, y, w, height, margin)
		next, effect := stepHover(h.hover, HoverSample{
			Now:          now,
			Inside:       inside,
			Outside:      outside,
			Delay:        cfg.HoverHideDelay(),
			RestoreDelay: cfg.HoverHideRestoreDelay(),
			Blocked:      h.hoverBlock(false),
			ButtonDown:   buttons,
			Enabled:      cfg.HoverHide,
			CardsShown:   shown,
		})
		h.hover = next
		if effect.Hide || effect.Restore {
			a.prepareCardOpacity(h)
			gen := h.beginHoverFade()
			applyHoverFade(windowAlphaFader{h.win}, effect, h.windowOpacity(), hoverFadeDuration)
			time.AfterFunc(hoverFadeDuration, func() {
				mygo.RunOnMain(func() { h.finishHoverFade(gen) })
			})
		}
	}
	a.ensureHoverPoll()
}
