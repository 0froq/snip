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
	roundContent(send(w, "contentView"), WindowCornerRadius)
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
