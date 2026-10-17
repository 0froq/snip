//go:build darwin

package macplace

import (
	"fmt"
	"math"
	"sync"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// NSSize must remain two float64s: on arm64 it is an HFA passed and returned
// in d0/d1, NOT an out pointer or a pair of integer registers. purego v0.11.1
// supports this ABI for both NewIMP callbacks and RegisterFunc calls.
type NSSize struct {
	Width, Height float64
}

var (
	resizeLoadOnce sync.Once
	setObjectClass func(id, objc.Class) objc.Class
	resizeClasses  = map[objc.Class]objc.Class{}
	resizeSizes    = map[id]func(int, int) (int, int){}
)

// ConstrainWindowResize installs content-bound native resizing. Call on the
// main thread, after creating the window, and call the returned cleanup on
// OnClosed. size receives the proposed width in points and returns the exact
// width and height; AppKit still owns edge tracking, cursors and frame origins.
func ConstrainWindowResize(handle uintptr, size func(int, int) (int, int)) func() {
	if handle == 0 {
		return func() {}
	}
	load()
	w := id(handle)
	delegate := send(w, "delegate")
	if delegate == 0 {
		panic("macplace: content resizing requires a window delegate")
	}
	cleanup := constrainDelegate(delegate, size)
	// NSWindow caches which optional delegate methods exist. Refresh that
	// cache now that this very same delegate responds to windowWillResize.
	send(w, "setDelegate:", 0)
	send(w, "setDelegate:", uintptr(delegate))
	return cleanup
}

func constrainDelegate(delegate id, size func(int, int) (int, int)) func() {
	resizeLoadOnce.Do(func() {
		purego.RegisterLibFunc(&setObjectClass, libObjC, "object_setClass")
	})
	if _, exists := resizeSizes[delegate]; exists {
		panic("macplace: resize delegate already installed")
	}
	base := delegate.Class()
	cls, ok := resizeClasses[base]
	if !ok {
		var err error
		// No ivars: changing isa is safe without changing the allocation size.
		// Inherit MyGo's methods and preserve self's identity: MyGo indexes its
		// windows by delegate ID, including for move/focus/resize/close and KVO.
		cls, err = objc.RegisterClass(fmt.Sprintf("SnipContentResizeDelegate_%x", uintptr(base)), base, nil, nil, []objc.MethodDef{{
			Cmd: objc.RegisterName("windowWillResize:toSize:"),
			Fn:  windowWillResize,
		}})
		if err != nil {
			panic("macplace: " + err.Error())
		}
		resizeClasses[base] = cls
	}
	// NSWindow's delegate is weak. Own an explicit ObjC retain until cleanup;
	// the map roots the Go closure (and its card state) independently of GC.
	send(delegate, "retain")
	resizeSizes[delegate] = size
	setObjectClass(delegate, cls)
	var closed bool
	return func() {
		if closed {
			return
		}
		closed = true
		delete(resizeSizes, delegate)
		setObjectClass(delegate, base)
		send(delegate, "release")
	}
}

func windowWillResize(self id, _ objc.SEL, window id, proposed NSSize) NSSize {
	// Only a live user drag picks the size. Programmatic frame changes (MyGo
	// constraints, SetSize) also pass through here and must not reset it.
	if !liveResize(window) && window != 0 && send(window, "respondsToSelector:", uintptr(sel("inLiveResize"))) != 0 {
		return proposed
	}
	if size := resizeSizes[self]; size != nil {
		w, h := size(int(math.Round(proposed.Width)), int(math.Round(proposed.Height)))
		return NSSize{Width: float64(w), Height: float64(h)}
	}
	return proposed
}

// InLiveResize reports whether the user is dragging the window's edge.
func InLiveResize(handle uintptr) bool {
	return liveResize(id(handle))
}

func liveResize(w id) bool {
	if w == 0 || send(w, "respondsToSelector:", uintptr(sel("inLiveResize"))) == 0 {
		return false
	}
	return sendInt(w, "inLiveResize")&0xff != 0
}
