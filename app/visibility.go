package app

import "github.com/0froq/snip/internal/macplace"

// showCard is the only note-show path. Call on the main thread, after sizing
// the window and updating any state (selection, pulse, etc.) its view needs.
func (a *App) showCard(h *cardUI) {
	if h == nil || h.win == nil {
		return
	}
	h.showWindow(h.win)
}

type noteWindow interface {
	ShowInactive()
	Invalidate()
	NativeHandle() uintptr
}

func (h *cardUI) showWindow(win noteWindow) {
	h.hideGen++ // Cancel any pending animated hide, including its posted callback.
	h.hiding = false
	win.ShowInactive()
	// Ordering a non-key window front does not invalidate MyGo's cached UI.
	// In particular, animated hide may have left a zero-opacity frame behind.
	// Ask MyGo to rebuild, then explicitly dirty the native view hierarchy so
	// rendering does not depend on activation, a click, or a display-link tick.
	win.Invalidate()
	macplace.DisplayWindow(win.NativeHandle())
}
