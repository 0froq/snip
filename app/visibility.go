package app

import (
	"github.com/0froq/snip/config"
	"github.com/0froq/snip/internal/macplace"
)

// cardsShown reports the global hide/show state.
func (a *App) cardsShown() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.visible
}

// showCard is the only note-show path. Call on the main thread, after sizing
// the window and updating any state (selection, pulse, etc.) its view needs.
// While hide-all is on, the card stays ordered out.
func (a *App) showCard(h *cardUI) {
	if h == nil || h.win == nil {
		return
	}
	// A show reveals the card. Drop any hover-hide so the window takes
	// mouse events again, without focusing it.
	h.hover = HoverState{}
	h.win.SetIgnoreMouseEvents(false)
	a.prepareCardOpacity(h)
	a.orderCardFront(h, h.showWindow)
}

// orderCardFront shows one card when the global state is shown. show is
// h.showWindow in production; tests pass a recorder.
func (a *App) orderCardFront(h *cardUI, show func(noteWindow)) {
	if h == nil || show == nil || !a.cardsShown() {
		return
	}
	show(h.win)
	a.syncBadge(h)
	// Showing a card orders it front. Lift every badge back above the cards,
	// then the which-key window above the badges.
	a.raiseLeaderChrome()
}

type noteWindow interface {
	ShowInactive()
	Invalidate()
	SetOpacity(float64)
	NativeHandle() uintptr
}

func (h *cardUI) showWindow(win noteWindow) {
	h.hideGen++ // Cancel any pending animated hide, including its posted callback.
	h.hiding = false
	h.orderedOut = false
	h.cancelHoverFade()
	// Paint the full note while the window is still ordered out (hide leaves
	// it at alpha 0) and only then order it in and reveal it at the
	// configured alpha. The hide path still drops alpha to 0. Stop a hover
	// fade first so it cannot land on top of that alpha.
	win.Invalidate()
	handle := win.NativeHandle()
	macplace.StopWindowAlphaAnimation(handle)
	macplace.DisplayWindow(handle)
	win.ShowInactive()
	win.SetOpacity(h.windowOpacity())
}

// prepareCardOpacity copies the configured alphas onto the card. Zero stays
// zero here; windowOpacity treats that as fully opaque.
func (a *App) prepareCardOpacity(h *cardUI) {
	if h == nil {
		return
	}
	c := a.config().Card
	h.opacityActive = c.ActiveOpacity
	h.opacityInactive = c.InactiveOpacity
}

// windowOpacity is the alpha for this card's selection. An unset config is 1.
func (h *cardUI) windowOpacity() float64 {
	return (config.Card{
		ActiveOpacity:   h.opacityActive,
		InactiveOpacity: h.opacityInactive,
	}).OpacityFor(h.selected)
}

// applyShownOpacity sets the window alpha of a card that is on screen.
// A card that hide ordered out stays at alpha 0 so the hide is not undone.
func (a *App) applyShownOpacity(h *cardUI) {
	if h == nil || h.win == nil || !opacityUpdateAllowed(h.hiding, h.orderedOut, h.hover.Hidden, h.hoverFading, a.cardsShown()) {
		return
	}
	a.prepareCardOpacity(h)
	h.win.SetOpacity(h.windowOpacity())
}
