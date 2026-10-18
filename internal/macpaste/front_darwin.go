//go:build darwin

package macpaste

import (
	"sync"
	"time"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

const (
	// NSApplicationActivateIgnoringOtherApps
	activateIgnoringOtherApps = 1 << 1
	frontWait                 = 300 * time.Millisecond
)

var (
	frontOnce    sync.Once
	frontFuncs   sync.Once
	dateSinceNow func(id, objc.SEL, float64) id
	runMode      func(id, objc.SEL, id, id) uint8
	objcRetain   uintptr
	objcRelease  uintptr
)

// WatchFrontmost records NSWorkspace's frontmost app, skipping this process.
// Call on the main thread before cards can take focus.
func WatchFrontmost() {
	frontOnce.Do(func() {
		load()
		fronts.setSelf(currentPID())
		if pid := frontPID(); pid > 0 {
			fronts.note(pid)
		}
		installFrontObserver()
	})
}

// PasteToPrevious resigns snip's key window, activates the last other app,
// and posts Cmd+V only after that app is frontmost.
func PasteToPrevious() error {
	load()
	if !AccessibilityTrusted(false) {
		return ErrAccessibility
	}
	WatchFrontmost()
	pid, ok := fronts.previous()
	if !ok {
		return ErrNoPreviousApp
	}
	activated := false
	withPool(func() {
		resignKeyWindow()
		app := send(class("NSRunningApplication"), "runningApplicationWithProcessIdentifier:", uintptr(int32(pid)))
		if app == 0 {
			return
		}
		retain(app)
		defer releaseObj(app)
		send(app, "activateWithOptions:", uintptr(activateIgnoringOtherApps))
		activated = waitUntilFront(pid, frontWait)
	})
	if !activated {
		return ErrPreviousAppInactive
	}
	return PostCommandV()
}

func currentPID() int {
	load()
	var pid int
	withPool(func() {
		pid = sendInt(send(class("NSProcessInfo"), "processInfo"), "processIdentifier")
	})
	return pid
}

func frontPID() int {
	app := send(send(class("NSWorkspace"), "sharedWorkspace"), "frontmostApplication")
	if app == 0 {
		return 0
	}
	return sendInt(app, "processIdentifier")
}

func resignKeyWindow() {
	app := send(class("NSApplication"), "sharedApplication")
	key := send(app, "keyWindow")
	if key == 0 {
		return
	}
	send(key, "makeFirstResponder:", 0)
	send(key, "resignKeyWindow")
}

func waitUntilFront(pid int, limit time.Duration) bool {
	deadline := time.Now().Add(limit)
	for {
		if frontPID() == pid {
			return true
		}
		if !time.Now().Before(deadline) {
			return frontPID() == pid
		}
		spinRunLoop(0.02)
	}
}

func spinRunLoop(seconds float64) {
	frontFuncs.Do(func() {
		purego.RegisterFunc(&dateSinceNow, msgSendAddr)
		purego.RegisterFunc(&runMode, msgSendAddr)
		objcRetain = dlsym(libObjC, "objc_retain")
		objcRelease = dlsym(libObjC, "objc_release")
	})
	when := dateSinceNow(class("NSDate"), sel("dateWithTimeIntervalSinceNow:"), seconds)
	loop := send(class("NSRunLoop"), "currentRunLoop")
	runMode(loop, sel("runMode:beforeDate:"), nsString("kCFRunLoopDefaultMode"), when)
}

func retain(obj id) {
	if obj == 0 {
		return
	}
	frontFuncs.Do(func() {
		purego.RegisterFunc(&dateSinceNow, msgSendAddr)
		purego.RegisterFunc(&runMode, msgSendAddr)
		objcRetain = dlsym(libObjC, "objc_retain")
		objcRelease = dlsym(libObjC, "objc_release")
	})
	_, _, _ = purego.SyscallN(objcRetain, uintptr(obj))
}

func releaseObj(obj id) {
	if obj == 0 || objcRelease == 0 {
		return
	}
	_, _, _ = purego.SyscallN(objcRelease, uintptr(obj))
}

func installFrontObserver() {
	cls, err := objc.RegisterClass("SnipFrontObserver", objc.GetClass("NSObject"), nil, nil, []objc.MethodDef{{
		Cmd: objc.RegisterName("snipAppActivated:"),
		Fn:  snipAppActivated,
	}})
	if err != nil {
		return
	}
	obs := send(send(id(cls), "alloc"), "init")
	ws := send(class("NSWorkspace"), "sharedWorkspace")
	center := send(ws, "notificationCenter")
	name := nsString("NSWorkspaceDidActivateApplicationNotification")
	send(center, "addObserver:selector:name:object:",
		uintptr(obs), uintptr(sel("snipAppActivated:")), uintptr(name), 0)
}

func snipAppActivated(_ id, _ objc.SEL, note id) {
	if note == 0 {
		return
	}
	info := send(note, "userInfo")
	if info == 0 {
		return
	}
	app := send(info, "objectForKey:", uintptr(nsString("NSWorkspaceApplicationKey")))
	if app == 0 {
		return
	}
	fronts.note(sendInt(app, "processIdentifier"))
}
