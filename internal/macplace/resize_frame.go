package macplace

// snapResizeFrame keeps a window's top and left edges while replacing its size.
// AppKit frames use a bottom-left origin, so the top edge is originY+height.
// A bottom-edge drag must not jump the card up when the height snaps to a
// whole line count.
func snapResizeFrame(originX, originY, _, height float64, newW, newH int) (x, y, w, h float64) {
	top := originY + height
	return originX, top - float64(newH), float64(newW), float64(newH)
}
