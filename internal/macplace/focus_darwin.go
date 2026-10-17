//go:build darwin

package macplace

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

var (
	focusLoadOnce sync.Once
	focusSetClass func(id, objc.Class) objc.Class
	nonKeyClasses = map[objc.Class]objc.Class{}
)

// PreventWindowFocus keeps a note from taking keyboard focus, even on clicks
// or native resize. Call on the main thread before first showing the window.
// The window must also be created as a nonactivating NSPanel: changing that
// style after creation does not reliably prevent app activation in AppKit.
func PreventWindowFocus(handle uintptr) {
	if handle == 0 {
		return
	}
	load()
	focusLoadOnce.Do(func() {
		purego.RegisterLibFunc(&focusSetClass, libObjC, "object_setClass")
	})
	w := id(handle)
	base := w.Class()
	cls, ok := nonKeyClasses[base]
	if !ok {
		var err error
		// No ivars or Go state: keep MyGo's window identity and inherit its
		// mouse handling and AppKit's native move/resize implementation.
		// Only these note instances change class, never Settings or MyGo's
		// shared panel class. The content view already accepts first mouse.
		cls, err = objc.RegisterClass(fmt.Sprintf("SnipNonKeyWindow_%x", uintptr(base)), base, nil, nil, []objc.MethodDef{
			{Cmd: objc.RegisterName("canBecomeKeyWindow"), Fn: cannotFocusWindow},
			{Cmd: objc.RegisterName("canBecomeMainWindow"), Fn: cannotFocusWindow},
		})
		if err != nil {
			panic("macplace: " + err.Error())
		}
		nonKeyClasses[base] = cls
		nonKeyClasses[cls] = cls // Idempotent for an already configured note.
	}
	focusSetClass(w, cls)
}

func cannotFocusWindow(id, objc.SEL) bool { return false }
