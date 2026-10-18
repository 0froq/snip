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
				rx, ry, _ := PlaceCard(ax, true, 0, 0, false, displays, cardW, cardH)
				result.x, result.y, result.ok = rx, ry, true
				return
			}
		}
		if mode == config.PlacementNearSelection || mode == config.PlacementNearCursor {
			cx, cy, okCur := cursorPoint()
			if okCur {
				rx, ry, _ := PlaceCard(Rect{}, false, cx, cy, true, displays, cardW, cardH)
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
	y, ok := FlipAppKitY(p.y, primaryScreenHeight())
	if !ok {
		return 0, 0, false
	}
	return int(p.x), y, true
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
	rectOnce  sync.Once
	readRect  func(id, objc.SEL) nsRect
	pointOnce sync.Once
	readPoint func(id, objc.SEL) nsPoint
	floatOnce sync.Once
	readFloat func(id, objc.SEL) float64
	stretOnce sync.Once
	stretAddr uintptr
)

// structReturnSend is the objc entry that returns an NSRect.
// arm64 has no objc_msgSend_stret: a four-double NSRect is an HFA in d0–d3
// and comes back through objc_msgSend. amd64 returns it through
// objc_msgSend_stret.
func structReturnSend() uintptr {
	load()
	if runtime.GOARCH != "amd64" {
		return msgSendAddr
	}
	stretOnce.Do(func() {
		stretAddr = dlsym(libObjC, "objc_msgSend_stret")
	})
	return stretAddr
}

// msgRect reads an NSRect-returning method.
//
// On arm64 the value is four float64s. RegisterFunc copies d0–d3 back into
// the struct. SyscallN does not: it only returns the integer registers, and
// the out-pointer sits in x2 while the indirect-result register x8 is left
// at 0. e310696 switched this helper to RegisterFunc only. That made
// NSWindow.frame visible to the resize snap, but a screen rect that the
// out-pointer had been returning then came back empty, and placement clamped
// the new card onto that empty rect. A positive out-pointer is therefore
// kept. A non-positive one is not the screen origin: the HFA read is used
// instead, which is what keeps a live-resize frame from being treated as
// (0,0). When both are empty the result stays empty so the caller can skip
// the write.
func msgRect(obj id, s objc.SEL) nsRect {
	load()
	if runtime.GOARCH == "arm64" {
		var indirect nsRect
		_, _, _ = purego.SyscallN(msgSendAddr, uintptr(obj), uintptr(s), uintptr(unsafe.Pointer(&indirect)))
		if usableFrame(indirect.origin.x, indirect.origin.y, indirect.size.width, indirect.size.height) {
			return indirect
		}
	}
	rectOnce.Do(func() {
		purego.RegisterFunc(&readRect, structReturnSend())
	})
	direct := readRect(obj, s)
	if usableFrame(direct.origin.x, direct.origin.y, direct.size.width, direct.size.height) {
		return direct
	}
	return nsRect{}
}

// msgPoint reads an NSPoint-returning method. Two float64s are an HFA in
// d0–d1 on arm64 and in xmm0/xmm1 on amd64. SyscallN leaves the Go value at
// (0,0), which is also the bottom-left of the primary screen. Once
// primaryScreenHeight() is a real height, flipping that failed point
// (y = height - 0) parks the cursor fallback on the bottom edge.
func msgPoint(obj id, s objc.SEL) nsPoint {
	load()
	if runtime.GOARCH == "arm64" {
		var indirect nsPoint
		_, _, _ = purego.SyscallN(msgSendAddr, uintptr(obj), uintptr(s), uintptr(unsafe.Pointer(&indirect)))
		if indirect.x != 0 || indirect.y != 0 {
			return indirect
		}
	}
	pointOnce.Do(func() {
		purego.RegisterFunc(&readPoint, msgSendAddr)
	})
	return readPoint(obj, s)
}

// msgFloat reads a CGFloat-returning method. On arm64 that value is in d0.
// SyscallN only returns the integer registers, so a scale or metric read
// through it stays 0.
func msgFloat(obj id, s objc.SEL) float64 {
	load()
	floatOnce.Do(func() {
		purego.RegisterFunc(&readFloat, msgSendAddr)
	})
	return readFloat(obj, s)
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
