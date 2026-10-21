//go:build darwin

package macplace

// DisplayWindow lays out and redraws a shown note without making it key or
// activating the application. Call on the main thread after invalidating the
// Go UI state; ordering an inactive panel front alone can leave its old frame.
func DisplayWindow(handle uintptr) {
	if handle == 0 {
		return
	}
	load()
	w := id(handle)
	view := send(w, "contentView")
	if view == 0 {
		return
	}
	send(view, "layoutSubtreeIfNeeded")
	invalidateViewTree(view)
	send(view, "displayIfNeeded")
}

func invalidateViewTree(view id) {
	send(view, "setNeedsDisplay:", 1)
	// MyGo draws in a layer-backed descendant, not necessarily contentView.
	// Dirty every view so that its updateLayer runs even on a non-key panel.
	children := send(view, "subviews")
	for i := range sendInt(children, "count") {
		invalidateViewTree(send(children, "objectAtIndex:", uintptr(i)))
	}
}
