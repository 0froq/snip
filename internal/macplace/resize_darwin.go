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
	setWindowFrame func(id, objc.SEL, nsRect, bool)
	setContentMin  func(id, objc.SEL, NSSize)
	contentMinSize func(id, objc.SEL) NSSize
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
	// cache now that this very same delegate responds to windowWillResize
	// and windowDidEndLiveResize.
	send(w, "setDelegate:", 0)
	send(w, "setDelegate:", uintptr(delegate))
	return cleanup
}

func constrainDelegate(delegate id, size func(int, int) (int, int)) func() {
	ensureResizeFuncs()
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
		cls, err = objc.RegisterClass(fmt.Sprintf("SnipContentResizeDelegate_%x", uintptr(base)), base, nil, nil, []objc.MethodDef{
			{
				Cmd: objc.RegisterName("windowWillResize:toSize:"),
				Fn:  windowWillResize,
			},
			{
				Cmd: objc.RegisterName("windowDidEndLiveResize:"),
				Fn:  windowDidEndLiveResize,
			},
		})
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
	// A non-positive proposal is never returned as-is: a fast drag can commit
	// a zero height while inLiveResize is already false, which hides the card.
	live := liveResize(window)
	nonPositive := proposed.Width <= 0 || proposed.Height <= 0
	if !live && !nonPositive && window != 0 && send(window, "respondsToSelector:", uintptr(sel("inLiveResize"))) != 0 {
		return proposed
	}
	minW, minH := windowMinContent(window)
	if size := resizeSizes[self]; size != nil {
		w, h := size(int(math.Round(proposed.Width)), int(math.Round(proposed.Height)))
		if w < minW {
			w = minW
		}
		if h < minH {
			h = minH
		}
		return NSSize{Width: float64(w), Height: float64(h)}
	}
	if proposed.Width < float64(minW) {
		proposed.Width = float64(minW)
	}
	if proposed.Height < float64(minH) {
		proposed.Height = float64(minH)
	}
	return proposed
}

// windowDidEndLiveResize runs once the user lets go. A fast drag can commit
// an in-between frame while inLiveResize is already false, so the positive
// path in windowWillResize leaves that size alone. Fit the frame that stuck,
// then clamp it to the content minimum and back onto a visibleFrame.
// inLiveResize is false here, so the correction's windowWillResize does not
// fit again.
func windowDidEndLiveResize(self id, _ objc.SEL, note id) {
	if note == 0 {
		return
	}
	window := send(note, "object")
	size := resizeSizes[self]
	if window == 0 || size == nil {
		return
	}
	frame := msgRect(window, sel("frame"))
	curW := int(math.Round(frame.size.width))
	curH := int(math.Round(frame.size.height))
	w, h := size(curW, curH)
	minW, minH := windowMinContent(window)
	if w < minW {
		w = minW
	}
	if h < minH {
		h = minH
	}
	visible := nearestVisible(screenVisibleFrames(), frame.origin.x, frame.origin.y, frame.size.width, frame.size.height)
	x, y, fw, fh := snapResizeFrame(frame.origin.x, frame.origin.y, frame.size.width, frame.size.height, w, h, minW, minH, visible)
	if x == frame.origin.x && y == frame.origin.y && fw == frame.size.width && fh == frame.size.height {
		return
	}
	setWindowFrame(window, sel("setFrame:display:"), nsRect{
		origin: nsPoint{x: x, y: y},
		size:   nsSize{width: fw, height: fh},
	}, true)
}

// SetContentMinSize sets the content view's minimum size. Borderless windows
// ignore the frame minSize after the style mask is cleared, so AppKit will
// otherwise collapse the content height to zero during a live resize.
func SetContentMinSize(handle uintptr, w, h int) {
	if handle == 0 {
		return
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	ensureResizeFuncs()
	setContentMin(id(handle), sel("setContentMinSize:"), NSSize{Width: float64(w), Height: float64(h)})
}

func ensureResizeFuncs() {
	load()
	resizeLoadOnce.Do(func() {
		purego.RegisterLibFunc(&setObjectClass, libObjC, "object_setClass")
		// NSRect is a 4-double HFA on arm64. RegisterFunc places it in d0–d3;
		// a raw SyscallN would pass it as integer registers or a pointer.
		// NSSize is the same ABI in d0/d1.
		purego.RegisterFunc(&setWindowFrame, msgSendAddr)
		purego.RegisterFunc(&setContentMin, msgSendAddr)
		purego.RegisterFunc(&contentMinSize, msgSendAddr)
	})
}

func windowMinContent(window id) (int, int) {
	if window == 0 || send(window, "respondsToSelector:", uintptr(sel("contentMinSize"))) == 0 {
		return 1, 1
	}
	ensureResizeFuncs()
	s := contentMinSize(window, sel("contentMinSize"))
	w := int(math.Round(s.Width))
	h := int(math.Round(s.Height))
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

// screenVisibleFrames reads each NSScreen.visibleFrame in AppKit's bottom-left
// space. Do not flip these through rectFromMac: snapResizeFrame compares them
// with the window frame, which uses the same origin.
func screenVisibleFrames() []box {
	screens := send(class("NSScreen"), "screens")
	if screens == 0 {
		return nil
	}
	n := sendInt(screens, "count")
	out := make([]box, 0, n)
	for i := 0; i < n; i++ {
		r := msgRect(send(screens, "objectAtIndex:", uintptr(i)), sel("visibleFrame"))
		if r.size.width <= 0 || r.size.height <= 0 {
			continue
		}
		out = append(out, box{
			X: r.origin.x, Y: r.origin.y,
			W: r.size.width, H: r.size.height,
		})
	}
	return out
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
