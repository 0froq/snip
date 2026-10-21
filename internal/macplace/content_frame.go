package macplace

const (
	styleBitTitled          = 1 << 0
	styleBitClosable        = 1 << 1
	styleBitMiniaturizable  = 1 << 2
	styleBitResizable       = 1 << 3
	styleBitFullSizeContent = 1 << 15
)

// borderlessStyleMask drops the titled window chrome, including
// fullSizeContentView. That bit is what keeps AppKit's hairline border
// after the title bar itself is hidden. The resizable bit stays so a
// borderless window can still track its edges.
func borderlessStyleMask(mask uintptr) uintptr {
	mask &^= styleBitTitled | styleBitClosable | styleBitMiniaturizable | styleBitFullSizeContent
	mask |= styleBitResizable
	return mask
}

// subviewShouldFill reports that a subview is the surface that used to
// fill the content view, now short by a title bar. A text view inset on
// the sides does not match: its width is not the content width.
func subviewShouldFill(contentW, contentH, x, y, w, h float64) bool {
	if contentW <= 0 || contentH <= 0 || w <= 0 || h <= 0 {
		return false
	}
	if x == 0 && y == 0 && w == contentW && h == contentH {
		return false
	}
	short := contentH - h
	return x == 0 && y >= 0 && w == contentW && h < contentH && short < 80
}

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
