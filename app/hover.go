package app

import (
	"time"

	"github.com/0froq/snip/internal/macplace"
	"github.com/egoist/mygo"
)

// hoverPollInterval is how often the global cursor is sampled. A hidden
// card receives no mouse events, and a pointer that rests still does not
// generate moves, so the dwell has to be counted from this poll.
const hoverPollInterval = 80 * time.Millisecond

// HoverState is one card's dwell. InsideSince is when the pointer entered
// the frame. It stays put while the pointer moves inside that frame.
// Hidden means the card has been dropped to alpha 0 and ignores the mouse.
type HoverState struct {
	InsideSince time.Time
	Hidden      bool
}

// HoverSample is one poll of a card. Blocked is drag, resize, text editing,
// an open context menu, or a mouse button held down.
type HoverSample struct {
	Now        time.Time
	Inside     bool
	Delay      time.Duration
	Blocked    bool
	Enabled    bool
	CardsShown bool
}

// HoverEffect is what the poll should do to the window. Hide and Restore
// are never both set. Neither means the window is left alone, so a hide-all
// that already ordered the card out is not shown again.
type HoverEffect struct {
	Hide    bool
	Restore bool
}

// stepHover advances a card's dwell. Time inside the frame counts whether
// or not the pointer is moving. Leaving the frame restores a hidden card.
// Hide-all (CardsShown false) clears the dwell and does not restore.
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
	if in.Blocked {
		if s.Hidden && !in.Inside {
			return HoverState{}, HoverEffect{Restore: true}
		}
		if s.Hidden {
			return HoverState{Hidden: true}, HoverEffect{}
		}
		return HoverState{}, HoverEffect{}
	}
	if !in.Inside {
		if s.Hidden {
			return HoverState{}, HoverEffect{Restore: true}
		}
		return HoverState{}, HoverEffect{}
	}
	if s.Hidden {
		return HoverState{Hidden: true}, HoverEffect{}
	}
	since := s.InsideSince
	if since.IsZero() {
		since = in.Now
	}
	if in.Delay > 0 && !in.Now.Before(since.Add(in.Delay)) {
		return HoverState{Hidden: true, InsideSince: since}, HoverEffect{Hide: true}
	}
	return HoverState{InsideSince: since}, HoverEffect{}
}

// pointerBlocksHover reports the conditions that must not start a hover hide.
func pointerBlocksHover(dragging, resizing, editing, menu, buttonDown bool) bool {
	return dragging || resizing || editing || menu || buttonDown
}

// pointInFrame reports whether a top-left screen point lies in a window frame.
// The right and bottom edges are outside, so leaving through that edge restores.
func pointInFrame(px, py, x, y, w, h int) bool {
	if w <= 0 || h <= 0 {
		return false
	}
	return px >= x && py >= y && px < x+w && py < y+h
}

// opacityUpdateAllowed is false while a hide, hide-all, or hover-hide owns
// the window alpha.
func opacityUpdateAllowed(hiding, orderedOut, hoverHidden, cardsShown bool) bool {
	return cardsShown && !hiding && !orderedOut && !hoverHidden
}

// hoverWindow is the slice of a card window hover-hide is allowed to touch.
// It has no Show or Focus, so a restore cannot take keyboard focus.
type hoverWindow interface {
	SetOpacity(float64)
	SetIgnoreMouseEvents(bool)
}

// applyHoverEffect hides by ignoring the mouse and dropping alpha, or
// restores the configured alpha and takes mouse events again.
func applyHoverEffect(win hoverWindow, effect HoverEffect, opacity float64) {
	if win == nil {
		return
	}
	if effect.Hide {
		win.SetIgnoreMouseEvents(true)
		win.SetOpacity(0)
		return
	}
	if effect.Restore {
		win.SetIgnoreMouseEvents(false)
		win.SetOpacity(opacity)
	}
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
		inside := pointInFrame(pt.X, pt.Y, x, y, w, height)
		next, effect := stepHover(h.hover, HoverSample{
			Now:        now,
			Inside:     inside,
			Delay:      cfg.HoverHideDelay(),
			Blocked:    h.hoverBlock(buttons),
			Enabled:    cfg.HoverHide,
			CardsShown: shown,
		})
		h.hover = next
		if effect.Hide || effect.Restore {
			a.prepareCardOpacity(h)
			applyHoverEffect(h.win, effect, h.windowOpacity())
		}
	}
	a.ensureHoverPoll()
}
