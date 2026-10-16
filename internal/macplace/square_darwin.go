//go:build darwin

package macplace

// SquareWindow clears the titled style bits that make AppKit round the frame,
// and turns the window shadow off so the slip's border is the window edge.
func SquareWindow(handle uintptr) {
	if handle == 0 {
		return
	}
	load()
	w := id(handle)
	const (
		styleTitled    = 1 << 0
		styleClosable  = 1 << 1
		styleMini      = 1 << 2
		styleResizable = 1 << 3
	)
	mask := uintptr(send(w, "styleMask"))
	mask &^= styleTitled | styleClosable | styleMini
	if mask&^styleResizable == 0 {
		mask |= styleResizable
	}
	send(w, "setStyleMask:", mask)
	send(w, "setHasShadow:", 0)
	send(w, "setOpaque:", 1)
	send(w, "setTitlebarAppearsTransparent:", 1)
	send(w, "setTitleVisibility:", 1)
}
