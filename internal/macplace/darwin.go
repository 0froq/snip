//go:build darwin

package macplace

import (
	"runtime"
	"sync"
	"unsafe"

	"github.com/0froq/snip/config"
	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// Supported is true on macOS builds.
const Supported = true

type id = objc.ID

var (
	loadOnce    sync.Once
	msgSendAddr uintptr
	poolPushFn  uintptr
	poolPopFn   uintptr
	selCache    = map[string]objc.SEL{}
	clsCache    = map[string]id{}
	libObjC     uintptr
	libAppKit   uintptr
	libAppServ  uintptr
	libCF       uintptr

	axUIElementCreateApplication          func(pid int32) uintptr
	axUIElementCopyAttributeValue         func(el uintptr, attr uintptr, out *uintptr) int32
	axUIElementCopyParameterizedAttribute func(el uintptr, attr uintptr, param uintptr, out *uintptr) int32
	axUIElementSetMessagingTimeout        func(el uintptr, timeout float32)
	axValueGetValue                       func(val uintptr, typ int32, out unsafe.Pointer) bool
	cfRelease                             func(cf uintptr)
)

const (
	nsUTF8StringEncoding = 4
	kAXErrorSuccess      = 0

	kAXValueCGRectType  = 3
	kAXValueCFRangeType = 4

	axMessagingTimeoutSec = 0.05
)

type cgPoint struct {
	x, y float64
}

type cgSize struct {
	width, height float64
}

type cgRect struct {
	origin cgPoint
	size   cgSize
}

// Origin picks a screen position for a new card using the configured placement mode.
func Origin(mode config.Placement, cardW, cardH int) (x, y int, ok bool) {
	if mode == config.PlacementFixed {
		return 0, 0, false
	}
	load()
	var result struct {
		x, y int
		ok   bool
	}
	withPool(func() {
		displays := readDisplays()
		if mode == config.PlacementNearSelection {
			if ax, okSel := selectionBounds(); okSel {
				wa := WorkAreaForPoint(ax.X+ax.Width/2, ax.Y+ax.Height/2, displays)
				rx, ry := PlaceBelowOrAbove(ax, cardW, cardH, wa)
				result.x, result.y, result.ok = rx, ry, true
				return
			}
		}
		if mode == config.PlacementNearSelection || mode == config.PlacementNearCursor {
			cx, cy, okCur := cursorPoint()
			if okCur {
				wa := WorkAreaForPoint(cx, cy, displays)
				rx, ry := PlaceNearCursor(cx, cy, cardW, cardH, wa)
				result.x, result.y, result.ok = rx, ry, true
			}
		}
	})
	return result.x, result.y, result.ok
}

func selectionBounds() (Rect, bool) {
	pid := frontmostPID()
	if pid == 0 {
		return Rect{}, false
	}
	app := axUIElementCreateApplication(pid)
	if app == 0 {
		return Rect{}, false
	}
	axUIElementSetMessagingTimeout(app, axMessagingTimeoutSec)

	focused := copyAttribute(app, "AXFocusedUIElement")
	if focused == 0 {
		return Rect{}, false
	}
	defer release(focused)

	rangeVal := copyAttribute(focused, "AXSelectedTextRange")
	if rangeVal == 0 {
		return Rect{}, false
	}
	defer release(rangeVal)

	boundsVal := copyParameterized(focused, "AXBoundsForRange", rangeVal)
	if boundsVal == 0 {
		return Rect{}, false
	}
	defer release(boundsVal)

	var r cgRect
	if !axValueGetValue(boundsVal, kAXValueCGRectType, unsafe.Pointer(&r)) {
		return Rect{}, false
	}
	rect := axCGRectToScreen(r)
	if rect.Width <= 0 || rect.Height <= 0 {
		return Rect{}, false
	}
	return rect, true
}

func axCGRectToScreen(r cgRect) Rect {
	// AX CGRect uses Quartz global display coordinates (origin top-left).
	return Rect{
		X:      int(r.origin.x),
		Y:      int(r.origin.y),
		Width:  int(r.size.width),
		Height: int(r.size.height),
	}
}

func frontmostPID() int32 {
	app := send(send(class("NSWorkspace"), "sharedWorkspace"), "frontmostApplication")
	if app == 0 {
		return 0
	}
	return int32(sendInt(app, "processIdentifier"))
}

func cursorPoint() (int, int, bool) {
	p := msgPoint(class("NSEvent"), sel("mouseLocation"))
	h := primaryScreenHeight()
	return int(p.x), int(h - p.y), true
}

func readDisplays() []Display {
	screens := send(class("NSScreen"), "screens")
	n := sendInt(screens, "count")
	out := make([]Display, 0, n)
	for i := 0; i < n; i++ {
		s := send(screens, "objectAtIndex:", uintptr(i))
		wa := rectFromMac(msgRect(s, sel("visibleFrame")))
		out = append(out, Display{WorkArea: wa})
	}
	return out
}

func copyAttribute(el uintptr, name string) uintptr {
	attr := nsString(name)
	var out uintptr
	if axUIElementCopyAttributeValue(el, uintptr(attr), &out) != kAXErrorSuccess || out == 0 {
		return 0
	}
	return out
}

func copyParameterized(el uintptr, name string, param uintptr) uintptr {
	attr := nsString(name)
	var out uintptr
	if axUIElementCopyParameterizedAttribute(el, uintptr(attr), param, &out) != kAXErrorSuccess || out == 0 {
		return 0
	}
	return out
}

func release(cf uintptr) {
	if cf != 0 {
		cfRelease(cf)
	}
}

func load() {
	loadOnce.Do(func() {
		libObjC = dlopen("/usr/lib/libobjc.A.dylib")
		libAppKit = dlopen("/System/Library/Frameworks/AppKit.framework/AppKit")
		libAppServ = dlopen("/System/Library/Frameworks/ApplicationServices.framework/ApplicationServices")
		libCF = dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation")

		msgSendAddr = dlsym(libObjC, "objc_msgSend")
		poolPushFn = dlsym(libObjC, "objc_autoreleasePoolPush")
		poolPopFn = dlsym(libObjC, "objc_autoreleasePoolPop")

		purego.RegisterLibFunc(&axUIElementCreateApplication, libAppServ, "AXUIElementCreateApplication")
		purego.RegisterLibFunc(&axUIElementCopyAttributeValue, libAppServ, "AXUIElementCopyAttributeValue")
		purego.RegisterLibFunc(&axUIElementCopyParameterizedAttribute, libAppServ, "AXUIElementCopyParameterizedAttributeValue")
		purego.RegisterLibFunc(&axUIElementSetMessagingTimeout, libAppServ, "AXUIElementSetMessagingTimeout")
		purego.RegisterLibFunc(&axValueGetValue, libAppServ, "AXValueGetValue")
		purego.RegisterLibFunc(&cfRelease, libCF, "CFRelease")
	})
}

func dlopen(path string) uintptr {
	h, err := purego.Dlopen(path, purego.RTLD_GLOBAL|purego.RTLD_NOW)
	if err != nil {
		panic("macplace: " + path + ": " + err.Error())
	}
	return h
}

func dlsym(lib uintptr, name string) uintptr {
	p, err := purego.Dlsym(lib, name)
	if err != nil {
		panic("macplace: " + name + ": " + err.Error())
	}
	return p
}

func sel(name string) objc.SEL {
	if s, ok := selCache[name]; ok {
		return s
	}
	s := objc.RegisterName(name)
	selCache[name] = s
	return s
}

func class(name string) id {
	if c, ok := clsCache[name]; ok {
		return c
	}
	c := id(objc.GetClass(name))
	if c == 0 {
		panic("macplace: class " + name + " missing")
	}
	clsCache[name] = c
	return c
}

//go:uintptrescapes
func send(obj id, selector string, args ...uintptr) id {
	s := sel(selector)
	var a [8]uintptr
	a[0], a[1] = uintptr(obj), uintptr(s)
	n := copy(a[2:], args)
	r, _, _ := purego.SyscallN(msgSendAddr, a[:n+2]...)
	return id(r)
}

func sendInt(obj id, selector string, args ...uintptr) int {
	return int(send(obj, selector, args...))
}

func withPool(fn func()) {
	load()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	pool, _, _ := purego.SyscallN(poolPushFn)
	defer purego.SyscallN(poolPopFn, pool)
	fn()
}

func nsString(s string) id {
	b := unsafe.StringData(s)
	str := send(send(class("NSString"), "alloc"), "initWithBytes:length:encoding:",
		uintptr(unsafe.Pointer(b)), uintptr(len(s)), nsUTF8StringEncoding)
	runtime.KeepAlive(s)
	return autorelease(str)
}

func autorelease(obj id) id {
	if obj == 0 {
		return 0
	}
	_, _, _ = purego.SyscallN(dlsym(libObjC, "objc_autorelease"), uintptr(obj))
	return obj
}

type nsPoint struct {
	x, y float64
}

type nsSize struct {
	width, height float64
}

type nsRect struct {
	origin nsPoint
	size   nsSize
}

var (
	rectOnce sync.Once
	readRect func(id, objc.SEL) nsRect
)

// msgRect reads an NSRect-returning method. On arm64 that struct is four
// float64s, an HFA returned in d0–d3. Passing an out-pointer through
// SyscallN leaves the Go value zero, and windowDidEndLiveResize then
// setFrame's the card to the bottom-left of the screen.
func msgRect(obj id, s objc.SEL) nsRect {
	load()
	rectOnce.Do(func() {
		purego.RegisterFunc(&readRect, msgSendAddr)
	})
	return readRect(obj, s)
}

func msgPoint(obj id, s objc.SEL) nsPoint {
	var p nsPoint
	_, _, _ = purego.SyscallN(msgSendAddr, uintptr(obj), uintptr(s), uintptr(unsafe.Pointer(&p)))
	return p
}

func primaryScreenHeight() float64 {
	screens := send(class("NSScreen"), "screens")
	if sendInt(screens, "count") == 0 {
		return 0
	}
	return msgRect(send(screens, "objectAtIndex:", 0), sel("frame")).size.height
}

func rectFromMac(r nsRect) Rect {
	h := primaryScreenHeight()
	return Rect{
		X:      int(r.origin.x),
		Y:      int(h - r.origin.y - r.size.height),
		Width:  int(r.size.width),
		Height: int(r.size.height),
	}
}
