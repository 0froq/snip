//go:build darwin

package macplace

import (
	"sync"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

var frameCursorOnce sync.Once

// UseFrameResizeCursors makes NSCursor's panel-style up/down and left/right
// resize cursors return the window-frame resize cursors (macOS 15+), so the
// widened in-content resize band shows the same cursor as the native window
// edge. Older macOS keeps AppKit's defaults.
func UseFrameResizeCursors() {
	frameCursorOnce.Do(func() {
		load()
		cls := objc.ID(objc.GetClass("NSCursor"))
		if cls == 0 || !responds(cls, "frameResizeCursorFromPosition:inDirections:") {
			return
		}
		var getClass func(objc.ID) objc.Class
		var replace func(objc.Class, objc.SEL, objc.IMP, string) objc.IMP
		purego.RegisterLibFunc(&getClass, libObjC, "object_getClass")
		purego.RegisterLibFunc(&replace, libObjC, "class_replaceMethod")
		meta := getClass(cls)
		// NSCursorFrameResizePosition: bottom = 1<<2, right = 1<<3.
		// NSCursorFrameResizeDirections: inward|outward = 3.
		frame := func(pos uintptr) func(objc.ID, objc.SEL) objc.ID {
			return func(objc.ID, objc.SEL) objc.ID {
				return send(cls, "frameResizeCursorFromPosition:inDirections:", pos, 3)
			}
		}
		replace(meta, objc.RegisterName("resizeUpDownCursor"), objc.NewIMP(frame(1<<2)), "@@:")
		replace(meta, objc.RegisterName("resizeLeftRightCursor"), objc.NewIMP(frame(1<<3)), "@@:")
	})
}

func responds(obj objc.ID, selector string) bool {
	return obj.Send(objc.RegisterName("respondsToSelector:"), objc.RegisterName(selector)) != 0
}
