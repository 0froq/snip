package app

import "github.com/0froq/snip/internal/macplace"

// showCard is the only note-show path. Call on the main thread, after sizing
// the window and updating any state (selection, pulse, etc.) its view needs.
func (a *App) showCard(h *cardUI) {
	if h == nil || h.win == nil {
		return
	}
	h.showWindow(h.win)
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
	// Paint the full note while the window is still ordered out (hide leaves
	// it at alpha 0) and only then order it in and reveal it.
	win.Invalidate()
	macplace.DisplayWindow(win.NativeHandle())
	win.ShowInactive()
	win.SetOpacity(1)
}
