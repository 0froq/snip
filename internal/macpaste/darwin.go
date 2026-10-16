//go:build darwin

package macpaste

import (
	"runtime"
	"sync"
	"unsafe"

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
	libCG       uintptr
	libAppKit   uintptr
	libAppServ  uintptr

	cgEventCreateKeyboardEvent func(source uintptr, virtualKey uint16, keyDown bool) uintptr
	cgEventSetFlags            func(event uintptr, flags uint64)
	cgEventPost                func(tap uint32, event uintptr)
	axIsProcessTrusted         func() uint8
	axIsProcessTrustedWithOpts func(options uintptr) uint8
)

const (
	utString                = "public.utf8-plain-text"
	nsUTF8StringEncoding    = 4
	kCGHIDEventTap          = 0
	kCGEventKeyDown         = 10
	kCGEventKeyUp           = 11
	kCGEventFlagMaskCommand = 1 << 20
	kVK_ANSI_V              = 0x09
)

func load() {
	loadOnce.Do(func() {
		libObjC = dlopen("/usr/lib/libobjc.A.dylib")
		libAppKit = dlopen("/System/Library/Frameworks/AppKit.framework/AppKit")
		libCG = dlopen("/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics")
		libAppServ = dlopen("/System/Library/Frameworks/ApplicationServices.framework/ApplicationServices")

		msgSendAddr = dlsym(libObjC, "objc_msgSend")
		poolPushFn = dlsym(libObjC, "objc_autoreleasePoolPush")
		poolPopFn = dlsym(libObjC, "objc_autoreleasePoolPop")

		purego.RegisterLibFunc(&cgEventCreateKeyboardEvent, libCG, "CGEventCreateKeyboardEvent")
		purego.RegisterLibFunc(&cgEventSetFlags, libCG, "CGEventSetFlags")
		purego.RegisterLibFunc(&cgEventPost, libCG, "CGEventPost")
		purego.RegisterLibFunc(&axIsProcessTrusted, libAppServ, "AXIsProcessTrusted")
		purego.RegisterLibFunc(&axIsProcessTrustedWithOpts, libAppServ, "AXIsProcessTrustedWithOptions")
	})
}

func dlopen(path string) uintptr {
	h, err := purego.Dlopen(path, purego.RTLD_GLOBAL|purego.RTLD_NOW)
	if err != nil {
		panic("macpaste: " + path + ": " + err.Error())
	}
	return h
}

func dlsym(lib uintptr, name string) uintptr {
	p, err := purego.Dlsym(lib, name)
	if err != nil {
		panic("macpaste: " + name + ": " + err.Error())
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
		panic("macpaste: class " + name + " missing")
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

func goString(str id) string {
	if str == 0 {
		return ""
	}
	p := send(str, "UTF8String")
	if p == 0 {
		return ""
	}
	n := sendInt(str, "lengthOfBytesUsingEncoding:", nsUTF8StringEncoding)
	return string(unsafe.Slice(*(**byte)(unsafe.Pointer(&p)), n))
}

func pasteboard() id {
	return send(class("NSPasteboard"), "generalPasteboard")
}

// ChangeCount reads NSPasteboard.changeCount.
func ChangeCount() int {
	load()
	var n int
	withPool(func() { n = sendInt(pasteboard(), "changeCount") })
	return n
}

// ReadPlainText reads UTF-8 plain text from the general pasteboard.
func ReadPlainText() string {
	load()
	var s string
	withPool(func() {
		s = goString(send(pasteboard(), "stringForType:", uintptr(nsString(utString))))
	})
	return s
}

// AccessibilityTrusted reports AX trust. The prompt option is passed only
// after a silent check fails, so a trusted app is never asked again.
func AccessibilityTrusted(prompt bool) bool {
	load()
	var trusted bool
	withPool(func() {
		trusted = axIsProcessTrusted() != 0 || axIsProcessTrustedWithOpts(0) != 0
		if trusted || !prompt {
			return
		}
		key := nsString("AXTrustedCheckOptionPrompt")
		num := send(class("NSNumber"), "numberWithBool:", uintptr(1))
		dict := send(class("NSDictionary"), "dictionaryWithObject:forKey:", uintptr(num), uintptr(key))
		trusted = axIsProcessTrustedWithOpts(uintptr(dict)) != 0
	})
	return trusted
}

// PostCommandV posts Cmd+V to the HID event tap.
func PostCommandV() error {
	load()
	if !AccessibilityTrusted(false) {
		return ErrAccessibility
	}
	withPool(func() {
		down := cgEventCreateKeyboardEvent(0, kVK_ANSI_V, true)
		up := cgEventCreateKeyboardEvent(0, kVK_ANSI_V, false)
		cgEventSetFlags(down, kCGEventFlagMaskCommand)
		cgEventSetFlags(up, kCGEventFlagMaskCommand)
		cgEventPost(kCGHIDEventTap, down)
		cgEventPost(kCGHIDEventTap, up)
	})
	return nil
}

// OpenAccessibilitySettings opens the Privacy > Accessibility pane (best effort).
func OpenAccessibilitySettings() {
	load()
	withPool(func() {
		url := nsString("x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility")
		ws := send(class("NSWorkspace"), "sharedWorkspace")
		send(ws, "openURL:", uintptr(send(class("NSURL"), "URLWithString:", uintptr(url))))
	})
}
