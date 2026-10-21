//go:build darwin

package macplace

import (
	"fmt"
	"log"
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
	resizeLoadOnce   sync.Once
	setObjectClass   func(id, objc.Class) objc.Class
	setWindowFrame   func(id, objc.SEL, nsRect, bool)
	setContentMin    func(id, objc.SEL, NSSize)
	contentMinSize   func(id, objc.SEL) NSSize
	setWindowAlpha   func(id, objc.SEL, float64)
	windowAlpha      func(id, objc.SEL) float64
	resizeClasses    = map[objc.Class]objc.Class{}
	resizeSizes      = map[id]func(int, int) (int, int){}
	frameWatchOnce   sync.Once
	frameWatchClass  objc.Class
	frameWatches     = map[id]*contentFrameWatch{}
	frameWatchByView = map[id]*contentFrameWatch{}
	windowRelayouts  = map[id]func(){}
)

// nsViewHeightSizable is AppKit's height-tracking bit. Width is already
// nsViewWidthSizable. A content view needs both, or a drag that does not
// move the window origin leaves the view at its old size.
const nsViewHeightSizable = 16

// contentFrameWatch is one card window. A borderless content view stays at
// (0, 0), so left and bottom drags are read off the window frame
// (windowDidResize / windowDidMove), not off the content view frame.
type contentFrameWatch struct {
	view, window               id
	lastX, lastY, lastW, lastH float64
	winX, winY, winW, winH     float64
	haveWin                    bool
	busy                       bool
	step                       string
	fn                         func()
}

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

// FollowContentFrame relayouts on a content-view frame change and on every
// live windowDidResize / windowDidMove. The content view of a borderless
// window stays at origin (0, 0), so a left or bottom drag never shows up
// there; the window frame is what moves. The view is made width- and
// height-sizable so a top or right drag still grows it. The same callback
// is the path for the setFrame at the end of a resize.
func FollowContentFrame(handle uintptr, relayout func()) func() {
	if handle == 0 || relayout == nil {
		return func() {}
	}
	load()
	ensureFrameWatch()
	window := id(handle)
	content := send(window, "contentView")
	if content == 0 {
		return func() {}
	}
	send(content, "setAutoresizingMask:", nsViewWidthSizable|nsViewHeightSizable)
	send(content, "setPostsFrameChangedNotifications:", 1)
	obs := send(send(id(frameWatchClass), "alloc"), "init")
	if obs == 0 {
		return func() {}
	}
	frame := msgRect(content, sel("frame"))
	winFrame := msgRect(window, sel("frame"))
	watch := &contentFrameWatch{
		view:    content,
		window:  window,
		lastX:   frame.origin.x,
		lastY:   frame.origin.y,
		lastW:   frame.size.width,
		lastH:   frame.size.height,
		winX:    winFrame.origin.x,
		winY:    winFrame.origin.y,
		winW:    winFrame.size.width,
		winH:    winFrame.size.height,
		haveWin: usableFrame(winFrame.origin.x, winFrame.origin.y, winFrame.size.width, winFrame.size.height),
		fn:      relayout,
	}
	frameWatches[obs] = watch
	frameWatchByView[content] = watch
	windowRelayouts[window] = relayout
	center := send(class("NSNotificationCenter"), "defaultCenter")
	send(center, "addObserver:selector:name:object:",
		uintptr(obs),
		uintptr(sel("snipContentViewFrameChanged:")),
		uintptr(nsString("NSViewFrameDidChangeNotification")),
		uintptr(content))
	// Posted for every live step, including a left or bottom drag whose
	// content view frame never changes.
	send(center, "addObserver:selector:name:object:",
		uintptr(obs),
		uintptr(sel("snipWindowLiveFrameChanged:")),
		uintptr(nsString("NSWindowDidResizeNotification")),
		uintptr(window))
	send(center, "addObserver:selector:name:object:",
		uintptr(obs),
		uintptr(sel("snipWindowLiveFrameChanged:")),
		uintptr(nsString("NSWindowDidMoveNotification")),
		uintptr(window))
	var closed bool
	return func() {
		if closed {
			return
		}
		closed = true
		send(send(class("NSNotificationCenter"), "defaultCenter"), "removeObserver:", uintptr(obs))
		delete(frameWatches, obs)
		delete(frameWatchByView, content)
		delete(windowRelayouts, window)
		send(obs, "release")
	}
}

// PinContentFrame puts the content view at (0,0) in the window, at the
// window's size. A bottom or left drag moves the window origin and can
// leave the view shifted inside it; the text then stays on the old bottom.
func PinContentFrame(handle uintptr) {
	if handle == 0 {
		return
	}
	load()
	window := id(handle)
	content := send(window, "contentView")
	if content == 0 {
		return
	}
	if watch := frameWatchByView[content]; watch != nil {
		pinContentView(watch)
		return
	}
	syncContentFrame(window, content)
}

func ensureFrameWatch() {
	frameWatchOnce.Do(func() {
		var err error
		frameWatchClass, err = objc.RegisterClass("SnipContentFrameWatch", objc.GetClass("NSObject"), nil, nil, []objc.MethodDef{
			{
				Cmd: objc.RegisterName("snipContentViewFrameChanged:"),
				Fn:  contentViewFrameChanged,
			},
			{
				Cmd: objc.RegisterName("snipWindowLiveFrameChanged:"),
				Fn:  windowLiveFrameChanged,
			},
		})
		if err != nil {
			panic("macplace: " + err.Error())
		}
	})
}

func contentViewFrameChanged(self id, _ objc.SEL, note id) {
	watch := frameWatches[self]
	if watch == nil || watch.fn == nil {
		return
	}
	view := watch.view
	if note != 0 {
		if object := send(note, "object"); object != 0 {
			view = object
		}
	}
	frame := msgRect(view, sel("frame"))
	x, y := frame.origin.x, frame.origin.y
	w, h := frame.size.width, frame.size.height
	if !contentFrameChanged(watch.lastX, watch.lastY, watch.lastW, watch.lastH, x, y, w, h) {
		return
	}
	watch.lastX, watch.lastY, watch.lastW, watch.lastH = x, y, w, h
	// Pin before the callback. The target is stored first, so the nested
	// setFrame sees the same frame and does not relayout again.
	watch.relayout("contentView")
}

// windowLiveFrameChanged is one live windowDidResize or windowDidMove.
// The content view origin is not consulted: it stays (0, 0) while the
// window origin moves.
func windowLiveFrameChanged(self id, _ objc.SEL, note id) {
	watch := frameWatches[self]
	if watch == nil || watch.fn == nil || watch.window == 0 || watch.busy {
		return
	}
	frame := msgRect(watch.window, sel("frame"))
	if !usableFrame(frame.origin.x, frame.origin.y, frame.size.width, frame.size.height) {
		return
	}
	live := liveResize(watch.window)
	needs := watch.haveWin && liveStepNeedsRelayout(live,
		watch.winX, watch.winY, watch.winW, watch.winH,
		frame.origin.x, frame.origin.y, frame.size.width, frame.size.height)
	watch.haveWin = true
	watch.winX, watch.winY = frame.origin.x, frame.origin.y
	watch.winW, watch.winH = frame.size.width, frame.size.height
	if !needs {
		return
	}
	watch.relayout(noteName(note))
}

func (watch *contentFrameWatch) relayout(path string) {
	if watch == nil || watch.fn == nil || watch.busy {
		return
	}
	watch.busy = true
	watch.step = path
	pinContentView(watch)
	// fn places the scroll view and, while the drag is live, displays and logs.
	watch.fn()
	watch.step = ""
	watch.busy = false
}

func noteName(note id) string {
	if note == 0 {
		return "live"
	}
	name := goString(send(note, "name"))
	if name == "" {
		return "live"
	}
	return name
}

// logLiveFrames prints the frames a live step left behind. The content
// origin stays (0, 0) on a borderless window. The scroll view is unflipped,
// so its top is y+height and has to track the window height.
func logLiveFrames(path string, window id) {
	if window == 0 {
		return
	}
	wf := msgRect(window, sel("frame"))
	var cf, sf nsRect
	if content := send(window, "contentView"); content != 0 {
		cf = msgRect(content, sel("frame"))
	}
	if h := cardTextForWindow(window); h != nil && h.scroll != 0 {
		sf = msgRect(h.scroll, sel("frame"))
	}
	log.Printf("snip: live step path=%s window=(%.1f,%.1f,%.1f,%.1f) content=(%.1f,%.1f,%.1f,%.1f) scroll=(%.1f,%.1f,%.1f,%.1f)",
		path,
		wf.origin.x, wf.origin.y, wf.size.width, wf.size.height,
		cf.origin.x, cf.origin.y, cf.size.width, cf.size.height,
		sf.origin.x, sf.origin.y, sf.size.width, sf.size.height)
}

// FinishLiveRelayout displays the text and logs the window, content, and
// scroll frames for a live step. A move that is not a live resize is quiet.
func FinishLiveRelayout(handle uintptr) {
	if handle == 0 || !liveResize(id(handle)) {
		return
	}
	window := id(handle)
	path := "relayout"
	if content := send(window, "contentView"); content != 0 {
		if watch := frameWatchByView[content]; watch != nil && watch.step != "" {
			path = watch.step
		}
	}
	displayLiveCard(window)
	logLiveFrames(path, window)
}

func displayLiveCard(window id) {
	if window == 0 {
		return
	}
	if h := cardTextForWindow(window); h != nil && h.scroll != 0 {
		send(h.scroll, "display")
	}
	if content := send(window, "contentView"); content != 0 {
		send(content, "displayIfNeeded")
	}
}

// pinContentView fills the window with the content view. The last frame is
// the pin target before setFrame, so the notification this setFrame posts
// is not a second relayout.
func pinContentView(watch *contentFrameWatch) {
	if watch == nil || watch.window == 0 || watch.view == 0 {
		return
	}
	win := msgRect(watch.window, sel("frame"))
	view := msgRect(watch.view, sel("frame"))
	x, y, w, h, changed := contentFrameMatchingWindow(
		win.size.width, win.size.height,
		view.origin.x, view.origin.y, view.size.width, view.size.height,
	)
	if !usableFrame(0, 0, w, h) {
		return
	}
	watch.lastX, watch.lastY, watch.lastW, watch.lastH = x, y, w, h
	if changed {
		syncContentFrame(watch.window, watch.view)
	}
	// Frame origin stays (0, 0) while a bottom drag shifts bounds.origin,
	// and an unflipped scroll view then rides the bottom edge.
	resetContentBounds(watch.view, w, h)
}

func resetContentBounds(view id, w, h float64) {
	if view == 0 || !usableFrame(0, 0, w, h) {
		return
	}
	frameOnce.Do(func() {
		purego.RegisterFunc(&setViewFrame, msgSendAddr)
		purego.RegisterFunc(&setLayerScale, msgSendAddr)
	})
	bounds := msgRect(view, sel("bounds"))
	if bounds.origin.x == 0 && bounds.origin.y == 0 && bounds.size.width == w && bounds.size.height == h {
		return
	}
	setViewFrame(view, sel("setBounds:"), nsRect{size: nsSize{width: w, height: h}})
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
		roundW := int(math.Round(proposed.Width))
		roundH := int(math.Round(proposed.Height))
		w, h := size(roundW, roundH)
		if w < minW {
			w = minW
		}
		if h < minH {
			h = minH
		}
		// A live drag must not receive a rounded size. AppKit anchors the
		// opposite edge from the size we return; rounding the free edge
		// moves that anchor and the next event moves it back.
		if live {
			nw, nh := constrainLiveResize(proposed.Width, proposed.Height, roundW, roundH, w, h)
			return NSSize{Width: nw, Height: nh}
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
	// A zero frame is a failed read, not a window at the screen origin.
	// Rebuilding from it setFrame's the card to the bottom-left.
	if !usableFrame(frame.origin.x, frame.origin.y, frame.size.width, frame.size.height) {
		log.Printf("snip: setFrame path=windowDidEndLiveResize skip invalid fromOrigin=(%g,%g) fromSize=(%g,%g)",
			frame.origin.x, frame.origin.y, frame.size.width, frame.size.height)
		return
	}
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
	screens := screenVisibleFrames()
	visible := nearestVisible(screens, frame.origin.x, frame.origin.y, frame.size.width, frame.size.height)
	x, y, fw, fh, apply := snapResizeFrame(frame.origin.x, frame.origin.y, frame.size.width, frame.size.height, w, h, minW, minH, visible)
	log.Printf("snip: setFrame path=windowDidEndLiveResize from=(%.1f,%.1f,%.1f,%.1f) snap=(%.1f,%.1f,%.1f,%.1f) fromOrigin=(%.1f,%.1f) newOrigin=(%.1f,%.1f) apply=%v onScreen=%v alpha=%v orderedIn=%v",
		frame.origin.x, frame.origin.y, frame.size.width, frame.size.height,
		x, y, fw, fh,
		frame.origin.x, frame.origin.y, x, y,
		apply,
		usableFrame(visible.X, visible.Y, visible.W, visible.H) && intersects(x, y, fw, fh, visible),
		readWindowAlpha(window), windowOrderedIn(window))
	if apply && usableFrame(x, y, fw, fh) &&
		(x != frame.origin.x || y != frame.origin.y || fw != frame.size.width || fh != frame.size.height) {
		setWindowFrame(window, sel("setFrame:display:"), nsRect{
			origin: nsPoint{x: x, y: y},
			size:   nsSize{width: fw, height: fh},
		}, true)
	} else if !apply || !usableFrame(x, y, fw, fh) {
		log.Printf("snip: reject setFrame path=windowDidEndLiveResize fromOrigin=(%g,%g) newOrigin=(%g,%g) rect=(%g,%g,%g,%g)",
			frame.origin.x, frame.origin.y, x, y, x, y, fw, fh)
	}
	// Live steps, this setFrame, and a setFrame that changes nothing share
	// one relayout. Pinning first keeps a bottom or left drag from leaving
	// the content view on the old origin; the callback places the text and
	// calls relayoutCard. A window that never installed the callback still
	// reflows the text that is already on it.
	if content := send(window, "contentView"); content != 0 {
		if watch := frameWatchByView[content]; watch != nil {
			pinContentView(watch)
		} else {
			syncContentFrame(window, content)
		}
	}
	if fn := windowRelayouts[window]; fn != nil {
		fn()
	} else if h := cardTextForWindow(window); h != nil {
		h.relayoutCard()
	}
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
		purego.RegisterFunc(&setWindowAlpha, msgSendAddr)
		purego.RegisterFunc(&windowAlpha, msgSendAddr)
	})
}

func readWindowAlpha(window id) float64 {
	if window == 0 || send(window, "respondsToSelector:", uintptr(sel("alphaValue"))) == 0 {
		return 1
	}
	ensureResizeFuncs()
	return windowAlpha(window, sel("alphaValue"))
}

func windowOrderedIn(window id) bool {
	if window == 0 || send(window, "respondsToSelector:", uintptr(sel("isVisible"))) == 0 {
		return true
	}
	return sendInt(window, "isVisible")&0xff != 0
}

// restoreResizeVisibility undoes an orderOut or alpha 0 that a resize
// callback managed to apply. A live resize must not leave the card hidden.
func restoreResizeVisibility(window id) {
	if window == 0 {
		return
	}
	if readWindowAlpha(window) == 0 && send(window, "respondsToSelector:", uintptr(sel("setAlphaValue:"))) != 0 {
		log.Printf("snip: resize path=windowDidEndLiveResize found alpha 0; restoring")
		ensureResizeFuncs()
		setWindowAlpha(window, sel("setAlphaValue:"), 1)
	}
	if !windowOrderedIn(window) {
		log.Printf("snip: resize path=windowDidEndLiveResize found orderOut; orderFrontRegardless")
		send(window, "orderFrontRegardless")
	}
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
