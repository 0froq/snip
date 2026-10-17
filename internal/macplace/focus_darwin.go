//go:build darwin

package macplace

// PreventWindowFocus keeps a note from taking keyboard focus, even on clicks
// or native resize. Call on the main thread before first showing the window.
// The window must also be created as a nonactivating NSPanel: changing that
// style after creation does not reliably prevent app activation in AppKit.
func PreventWindowFocus(handle uintptr) {
	if handle == 0 {
		return
	}
	load()
	w := id(handle)
	// Swapping the instance class (object_setClass) trips AppKit's
	// NSDynamicProperties cache and aborts on later setters such as
	// setStyleMask:. A nonactivating panel that only becomes key when a
	// control needs it gives the same behavior without subclassing.
	if send(w, "respondsToSelector:", uintptr(sel("setBecomesKeyOnlyIfNeeded:"))) != 0 {
		send(w, "setBecomesKeyOnlyIfNeeded:", 1)
	}
}
