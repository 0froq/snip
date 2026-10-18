//go:build darwin

package macplace

import (
	"sync"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

var (
	roundOnce      sync.Once
	setLayerRadius func(id, objc.SEL, float64)
	setLayerScale  func(id, objc.SEL, float64)
	frameOnce      sync.Once
	setViewFrame   func(id, objc.SEL, nsRect)
)

// SquareWindow clears the titled style bits that make AppKit use its own
// ~10–12pt corner radius, then clips the content view to WindowCornerRadius.
// The window is non-opaque so the clipped corners show through. The resizable
// bit stays on: borderless windows still need it for edge tracking and resize
// cursors. Hit testing uses the rectangular frame, so the resize band is unchanged.
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
	mask |= styleResizable
	send(w, "setStyleMask:", mask)
	send(w, "setHasShadow:", 0)
	send(w, "setOpaque:", 0)
	send(w, "setBackgroundColor:", uintptr(send(class("NSColor"), "clearColor")))
	send(w, "setTitlebarAppearsTransparent:", 1)
	send(w, "setTitleVisibility:", 1)
	content := send(w, "contentView")
	roundContent(content, WindowCornerRadius)
	// setStyleMask keeps the old content rect. The window frame and the
	// view the border is drawn into then disagree by about a title bar.
	// Read both rects as arm64 HFAs and make the content view fill the frame.
	syncContentFrame(w, content)
}

func syncContentFrame(w, content id) {
	if w == 0 || content == 0 {
		return
	}
	frame := msgRect(w, sel("frame"))
	view := msgRect(content, sel("frame"))
	x, y, fw, fh, changed := contentFrameMatchingWindow(
		frame.size.width, frame.size.height,
		view.origin.x, view.origin.y, view.size.width, view.size.height,
	)
	if !usableFrame(0, 0, fw, fh) {
		return
	}
	frameOnce.Do(func() {
		// NSRect arguments travel in d0–d3 on arm64. A SyscallN pointer
		// would not be the frame AppKit applies.
		purego.RegisterFunc(&setViewFrame, msgSendAddr)
		purego.RegisterFunc(&setLayerScale, msgSendAddr)
	})
	bounds := nsRect{size: nsSize{width: fw, height: fh}}
	if changed {
		setViewFrame(content, sel("setFrame:"), nsRect{origin: nsPoint{x: x, y: y}, size: bounds.size})
		subs := send(content, "subviews")
		n := sendInt(subs, "count")
		for i := 0; i < n; i++ {
			setViewFrame(send(subs, "objectAtIndex:", uintptr(i)), sel("setFrame:"), bounds)
		}
	}
	layer := send(content, "layer")
	if layer == 0 {
		return
	}
	setViewFrame(layer, sel("setBounds:"), bounds)
	if scale := msgFloat(w, sel("backingScaleFactor")); scale > 0 {
		setLayerScale(layer, sel("setContentsScale:"), scale)
	}
}

func roundContent(view id, radius float64) {
	if view == 0 {
		return
	}
	roundOnce.Do(func() {
		// CGFloat is a double in a floating-point register on arm64.
		purego.RegisterFunc(&setLayerRadius, msgSendAddr)
	})
	send(view, "setWantsLayer:", 1)
	layer := send(view, "layer")
	if layer == 0 {
		return
	}
	setLayerRadius(layer, sel("setCornerRadius:"), radius)
	send(layer, "setMasksToBounds:", 1)
}
