package macplace

// contentFrameMatchingWindow is the content-view frame that fills a borderless
// window. AppKit view frames are bottom-left inside the window. After the
// titled style is cleared, the content view can keep the old title-bar inset
// while the window frame does not, so the view the border is drawn in is
// shorter or shifted relative to the window. A failed window read (non-positive
// size) is left untouched.
func contentFrameMatchingWindow(frameW, frameH, viewX, viewY, viewW, viewH float64) (x, y, w, h float64, changed bool) {
	if !usableFrame(0, 0, frameW, frameH) {
		return viewX, viewY, viewW, viewH, false
	}
	if viewX == 0 && viewY == 0 && viewW == frameW && viewH == frameH {
		return 0, 0, frameW, frameH, false
	}
	return 0, 0, frameW, frameH, true
}
