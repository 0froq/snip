//go:build darwin

package macplace

import (
	"math"
	"sync"
	"unsafe"

	"github.com/0froq/snip/config"
	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// NativeCardText is true on macOS: the card hosts an NSScrollView whose
// document is an NSTextView. MyGo's text area is its own layout widget,
// not that pair, so the card cannot use it for native scrolling or editing.

const NativeCardText = true

const (
	nsViewWidthSizable     = 2
	nsEventModifierCommand = 1 << 20
	nsScrollerStyleOverlay = 1
	// Left, right, and other mouse-down. A global monitor only sees events
	// delivered to other apps; the local one sees events in this app.
	mouseDownMask = (1 << 1) | (1 << 3) | (1 << 25)
)

// CardText is the NSScrollView embedded in one card window.
type CardText struct {
	window   id
	scroll   id
	view     id
	delegate id

	editing bool
	closing bool
	canEdit bool
	titles  CardMenuTitles

	onAction func(CardMenuAction)
	onCommit func(string)

	localMonitor  id
	globalMonitor id
	localBlock    objc.Block
	globalBlock   objc.Block

	fontFamily  string
	fontSize    float64
	fontApplied bool
	red         uint8
	green       uint8
	blue        uint8
	colorSet    bool
	haveText    bool
	text        string

	// fadeMask is the clip view's alpha mask. Nil while the text fits.
	fadeMask     id
	fadeWatching bool
}

var (
	cardTextOnce sync.Once
	cardTextMu   sync.Mutex
	cardTexts    = map[id]*CardText{}

	classGetImp  func(objc.Class, objc.SEL) uintptr
	scrollHitImp func(id, objc.SEL, nsPoint) id
	initFrame    func(id, objc.SEL, nsRect) id
	fontNamed    func(id, objc.SEL, id, float64) id
	fontSystem   func(id, objc.SEL, float64) id
	colorRGBA    func(id, objc.SEL, float64, float64, float64, float64) id
	colorSRGB    func(id, objc.SEL, float64, float64, float64, float64) id
	numDouble    func(id, objc.SEL, float64) id
	setNSSize    func(id, objc.SEL, nsSize)
	setCGFloat   func(id, objc.SEL, float64)
	setCGPoint   func(id, objc.SEL, nsPoint)
	convertPt    func(id, objc.SEL, nsPoint, id) nsPoint
	popUpMenu    func(id, objc.SEL, id, nsPoint, id) bool
)

// AttachCardText adds a transparent NSScrollView to the window's content
// view. The text view is not editable until BeginEdit. handle 0 returns nil.
func AttachCardText(handle uintptr, opt CardTextOptions) *CardText {
	if handle == 0 {
		return nil
	}
	ensureCardText()
	h := &CardText{
		window:     id(handle),
		canEdit:    opt.CanEdit,
		titles:     opt.Titles,
		onAction:   opt.OnAction,
		onCommit:   opt.OnCommit,
		fontFamily: opt.FontFamily,
		fontSize:   opt.FontSize,
		red:        opt.Red,
		green:      opt.Green,
		blue:       opt.Blue,
		colorSet:   true,
	}
	x, y, w, ht, ok := TextContentFrame(opt.Width, opt.Height, opt.PadX, opt.PadY, opt.Border, opt.ResizeMargin, opt.TopExtra)
	if !ok {
		x, y, w, ht = 0, 0, 1, 1
	}
	frame := nsRect{origin: nsPoint{x: x, y: y}, size: nsSize{width: w, height: ht}}
	scroll := initFrame(send(id(scrollClass), "alloc"), sel("initWithFrame:"), frame)
	view := initFrame(send(class("NSTextView"), "alloc"), sel("initWithFrame:"), nsRect{size: frame.size})
	if scroll == 0 || view == 0 {
		return nil
	}
	delegate := send(send(id(delegateClass), "alloc"), "init")
	h.scroll = scroll
	h.view = view
	h.delegate = delegate

	send(scroll, "setDrawsBackground:", 0)
	send(scroll, "setBorderType:", 0)
	send(scroll, "setHasVerticalScroller:", 1)
	send(scroll, "setHasHorizontalScroller:", 0)
	send(scroll, "setAutohidesScrollers:", 1)
	// Overlay scrollers do not narrow the clip view. A legacy scroller
	// would, and the last word would wrap in a card sized to the ink.
	send(scroll, "setScrollerStyle:", nsScrollerStyleOverlay)
	send(scroll, "setAutomaticallyAdjustsContentInsets:", 0)
	send(scroll, "setAutoresizingMask:", 0)
	send(scroll, "setDocumentView:", uintptr(view))

	send(view, "setDelegate:", uintptr(delegate))
	send(view, "setDrawsBackground:", 0)
	send(view, "setBackgroundColor:", uintptr(send(class("NSColor"), "clearColor")))
	send(view, "setRichText:", 0)
	send(view, "setImportsGraphics:", 0)
	send(view, "setEditable:", 0)
	send(view, "setSelectable:", 0)
	send(view, "setVerticallyResizable:", 1)
	send(view, "setHorizontallyResizable:", 0)
	send(view, "setAutoresizingMask:", nsViewWidthSizable)
	if container := send(view, "textContainer"); container != 0 {
		send(container, "setWidthTracksTextView:", 1)
	}
	applyTextContainer(view, send(view, "textContainer"), w, ht)
	setNSSize(view, sel("setMaxSize:"), nsSize{width: math.MaxFloat64, height: math.MaxFloat64})

	h.writeText(opt.Text)
	cardTextMu.Lock()
	cardTexts[scroll] = h
	cardTexts[view] = h
	cardTexts[delegate] = h
	cardTextMu.Unlock()

	content := send(h.window, "contentView")
	send(content, "addSubview:", uintptr(scroll))
	h.installFades()
	return h
}

// Layout moves the scroll view to the inset content frame. A window that
// is too small to inset is left where it was. Editing keeps the draft.
func (h *CardText) Layout(winW, winH, padX, padY, border, resizeMargin, topExtra int) {
	if h == nil || h.closing || h.scroll == 0 {
		return
	}
	x, y, w, ht, ok := TextContentFrame(winW, winH, padX, padY, border, resizeMargin, topExtra)
	if !ok {
		return
	}
	frameOnce.Do(func() {
		purego.RegisterFunc(&setViewFrame, msgSendAddr)
		purego.RegisterFunc(&setLayerScale, msgSendAddr)
	})
	setViewFrame(h.scroll, sel("setFrame:"), nsRect{origin: nsPoint{x: x, y: y}, size: nsSize{width: w, height: ht}})
	applyTextContainer(h.view, send(h.view, "textContainer"), w, ht)
	h.updateFades()
}

// applyTextContainer writes the same insets the card height was measured
// with. textContainerInset is an NSSize (HFA). lineFragmentPadding is a
// CGFloat in d0, so it cannot go through SyscallN.
func applyTextContainer(view, container id, viewW, viewH float64) {
	if view == 0 {
		return
	}
	setNSSize(view, sel("setTextContainerInset:"), nsSize{
		width:  config.TextContainerInsetX,
		height: config.TextContainerInsetY,
	})
	if container != 0 {
		send(container, "setWidthTracksTextView:", 1)
		setCGFloat(container, sel("setLineFragmentPadding:"), config.LineFragmentPadding)
		inner := viewW - 2*config.TextContainerInsetX
		if inner < 1 {
			inner = 1
		}
		setNSSize(container, sel("setContainerSize:"), nsSize{width: inner, height: math.MaxFloat64})
	}
	setNSSize(view, sel("setMinSize:"), nsSize{width: 0, height: viewH})
}

// SetText replaces the shown string. A draft in progress is left alone,
// and writing the same string again does not reset the scroll position.
func (h *CardText) SetText(s string) {
	if h == nil || h.closing || h.editing {
		return
	}
	if h.haveText && h.text == s {
		return
	}
	h.writeText(s)
}

// SetFont applies the card face. A missing concrete family falls back to
// the system font at the same size.
func (h *CardText) SetFont(family string, size float64) {
	if h == nil || h.closing {
		return
	}
	if h.fontApplied && h.fontFamily == family && h.fontSize == size {
		return
	}
	h.fontFamily = family
	h.fontSize = size
	h.applyFont()
}

// SetColor is the text color, in sRGB bytes.
func (h *CardText) SetColor(r, g, b uint8) {
	if h == nil || h.closing || h.view == 0 {
		return
	}
	if h.colorSet && h.red == r && h.green == g && h.blue == b {
		return
	}
	h.red, h.green, h.blue = r, g, b
	h.colorSet = true
	h.applyColor()
}

// SetVisible shows or hides the scroll view. A pixel-font card hides it
// so the bitmap text underneath receives the clicks.
func (h *CardText) SetVisible(on bool) {
	if h == nil || h.closing || h.scroll == 0 {
		return
	}
	hidden := uintptr(1)
	if on {
		hidden = 0
	}
	send(h.scroll, "setHidden:", hidden)
}

// SetCanEdit hides Edit on the next menu when the card is sensitive.
func (h *CardText) SetCanEdit(can bool) {
	if h == nil {
		return
	}
	h.canEdit = can
}

// SetTitles updates the context-menu labels.
func (h *CardText) SetTitles(titles CardMenuTitles) {
	if h == nil {
		return
	}
	h.titles = titles
}

// BeginEdit shows raw and makes the text view the field editor.
// A sensitive card (canEdit false) does not enter edit mode.
func (h *CardText) BeginEdit(raw string) {
	if h == nil || h.closing || h.view == 0 || !h.canEdit || h.editing {
		return
	}
	h.writeText(raw)
	send(h.view, "setSelectable:", 1)
	send(h.view, "setEditable:", 1)
	h.installMonitors()
	h.editing = true
	if send(h.window, "makeFirstResponder:", uintptr(h.view)) == 0 {
		send(h.window, "makeKeyWindow")
		send(h.window, "makeFirstResponder:", uintptr(h.view))
	}
}

// CancelEdit leaves the draft without saving it.
func (h *CardText) CancelEdit() {
	if h == nil || !h.editing {
		return
	}
	h.editing = false
	h.removeMonitors()
	if h.view != 0 {
		send(h.view, "setEditable:", 0)
		send(h.view, "setSelectable:", 0)
	}
}

// Editing reports whether a draft is open.
func (h *CardText) Editing() bool {
	return h != nil && h.editing
}

// Text is the string currently in the text view.
func (h *CardText) Text() string {
	if h == nil || h.view == 0 {
		return ""
	}
	return goString(send(h.view, "string"))
}

// ShowMenu opens the context menu at a top-left window point.
func (h *CardText) ShowMenu(x, y, winW, winH int) {
	if h == nil || h.closing || h.window == 0 || winW < 1 || winH < 1 {
		return
	}
	content := send(h.window, "contentView")
	if content == 0 {
		return
	}
	menu := h.buildMenu()
	pt := nsPoint{x: float64(x), y: float64(winH - y)}
	popUpMenu(menu, sel("popUpMenuPositioningItem:atLocation:inView:"), 0, pt, content)
	send(menu, "release")
}

// Close removes the scroll view and its monitors. An open draft is not saved.
func (h *CardText) Close() {
	if h == nil || h.closing {
		return
	}
	h.closing = true
	h.editing = false
	h.removeMonitors()
	h.removeFadeWatch()
	h.onAction = nil
	h.onCommit = nil
	cardTextMu.Lock()
	delete(cardTexts, h.scroll)
	delete(cardTexts, h.view)
	delete(cardTexts, h.delegate)
	cardTextMu.Unlock()
	if h.scroll != 0 {
		send(h.scroll, "removeFromSuperview")
	}
	if h.delegate != 0 {
		send(h.delegate, "release")
	}
}

func (h *CardText) commit() {
	if h == nil || h.closing || !h.editing {
		return
	}
	h.editing = false
	text := h.Text()
	h.haveText = true
	h.text = text
	h.removeMonitors()
	if h.view != 0 {
		send(h.view, "setEditable:", 0)
		send(h.view, "setSelectable:", 0)
	}
	if h.onCommit != nil {
		h.onCommit(text)
	}
}

func (h *CardText) writeText(s string) {
	if h.view == 0 {
		return
	}
	send(h.view, "setString:", uintptr(nsString(s)))
	h.haveText = true
	h.text = s
	h.fontApplied = false
	h.applyFont()
	h.updateFades()
}

func (h *CardText) applyFont() {
	if h.view == 0 {
		return
	}
	size := h.fontSize
	if size <= 0 {
		size = 14
	}
	var font id
	if name, system := FontFace(h.fontFamily); !system && name != "" {
		font = fontNamed(class("NSFont"), sel("fontWithName:size:"), nsString(name), size)
	}
	if font == 0 {
		font = fontSystem(class("NSFont"), sel("systemFontOfSize:"), size)
	}
	if font != 0 {
		send(h.view, "setFont:", uintptr(font))
	}
	h.fontApplied = true
	if h.colorSet {
		h.applyColor()
	}
	h.updateFades()
}

func (h *CardText) applyColor() {
	if h.view == 0 {
		return
	}
	c := colorRGBA(class("NSColor"), sel("colorWithCalibratedRed:green:blue:alpha:"),
		float64(h.red)/255, float64(h.green)/255, float64(h.blue)/255, 1)
	if c != 0 {
		send(h.view, "setTextColor:", uintptr(c))
	}
}

func (h *CardText) buildMenu() id {
	menu := send(send(class("NSMenu"), "alloc"), "init")
	for _, item := range CardMenuItems(h.canEdit, h.titles) {
		mi := send(send(class("NSMenuItem"), "alloc"), "initWithTitle:action:keyEquivalent:",
			uintptr(nsString(item.Title)), uintptr(sel("snipMenuAction:")), uintptr(nsString("")))
		send(mi, "setTarget:", uintptr(h.delegate))
		send(mi, "setTag:", uintptr(item.Action))
		send(menu, "addItem:", uintptr(mi))
		send(mi, "release")
	}
	return menu
}

func (h *CardText) installMonitors() {
	if h.localMonitor != 0 || h.globalMonitor != 0 {
		return
	}
	h.localBlock = objc.NewBlock(func(_ objc.Block, event objc.ID) objc.ID {
		h.onMouse(id(event), false)
		return event
	})
	h.globalBlock = objc.NewBlock(func(_ objc.Block, event objc.ID) {
		h.onMouse(id(event), true)
	})
	h.localMonitor = send(class("NSEvent"), "addLocalMonitorForEventsMatchingMask:handler:",
		mouseDownMask, uintptr(h.localBlock))
	h.globalMonitor = send(class("NSEvent"), "addGlobalMonitorForEventsMatchingMask:handler:",
		mouseDownMask, uintptr(h.globalBlock))
}

func (h *CardText) removeMonitors() {
	if h.localMonitor != 0 {
		send(class("NSEvent"), "removeMonitor:", uintptr(h.localMonitor))
		h.localMonitor = 0
	}
	if h.globalMonitor != 0 {
		send(class("NSEvent"), "removeMonitor:", uintptr(h.globalMonitor))
		h.globalMonitor = 0
	}
	if h.localBlock != 0 {
		h.localBlock.Release()
		h.localBlock = 0
	}
	if h.globalBlock != 0 {
		h.globalBlock.Release()
		h.globalBlock = 0
	}
}

func (h *CardText) onMouse(event id, global bool) {
	if h.closing || !h.editing {
		return
	}
	if global || event == 0 {
		h.commit()
		return
	}
	if send(event, "window") != h.window {
		h.commit()
		return
	}
	pt := msgPoint(event, sel("locationInWindow"))
	local := convertPt(h.scroll, sel("convertPoint:fromView:"), pt, 0)
	bounds := msgRect(h.scroll, sel("bounds"))
	if !pointInNSRect(local.x, local.y, bounds.origin.x, bounds.origin.y, bounds.size.width, bounds.size.height) {
		h.commit()
	}
}

var (
	scrollClass   objc.Class
	delegateClass objc.Class
)

func ensureCardText() {
	load()
	cardTextOnce.Do(func() {
		dlopen("/System/Library/Frameworks/QuartzCore.framework/QuartzCore")
		purego.RegisterLibFunc(&classGetImp, libObjC, "class_getMethodImplementation")
		if imp := classGetImp(objc.GetClass("NSScrollView"), sel("hitTest:")); imp != 0 {
			purego.RegisterFunc(&scrollHitImp, imp)
		}
		purego.RegisterFunc(&initFrame, msgSendAddr)
		purego.RegisterFunc(&fontNamed, msgSendAddr)
		purego.RegisterFunc(&fontSystem, msgSendAddr)
		purego.RegisterFunc(&colorRGBA, msgSendAddr)
		purego.RegisterFunc(&colorSRGB, msgSendAddr)
		purego.RegisterFunc(&numDouble, msgSendAddr)
		purego.RegisterFunc(&setNSSize, msgSendAddr)
		purego.RegisterFunc(&setCGPoint, msgSendAddr)
		purego.RegisterFunc(&setCGFloat, msgSendAddr)
		purego.RegisterFunc(&convertPt, msgSendAddr)
		purego.RegisterFunc(&popUpMenu, msgSendAddr)
		frameOnce.Do(func() {
			purego.RegisterFunc(&setViewFrame, msgSendAddr)
			purego.RegisterFunc(&setLayerScale, msgSendAddr)
		})

		var err error
		scrollClass, err = objc.RegisterClass("SnipCardScrollView", objc.GetClass("NSScrollView"), nil, nil, []objc.MethodDef{{
			Cmd: objc.RegisterName("hitTest:"),
			Fn:  snipScrollHitTest,
		}})
		if err != nil {
			panic("macplace: " + err.Error())
		}
		delegateClass, err = objc.RegisterClass("SnipCardTextDelegate", objc.GetClass("NSObject"), nil, nil, []objc.MethodDef{
			{Cmd: objc.RegisterName("textView:doCommandBySelector:"), Fn: cardDoCommand},
			{Cmd: objc.RegisterName("textDidEndEditing:"), Fn: cardDidEndEditing},
			{Cmd: objc.RegisterName("textDidChange:"), Fn: cardTextDidChange},
			{Cmd: objc.RegisterName("textView:menu:forEvent:atIndex:"), Fn: cardMenuForEvent},
			{Cmd: objc.RegisterName("snipMenuAction:"), Fn: cardMenuAction},
			{Cmd: objc.RegisterName("snipClipBoundsChanged:"), Fn: cardClipBoundsChanged},
		})
		if err != nil {
			panic("macplace: " + err.Error())
		}
	})
}

func cardTextDidChange(self id, _ objc.SEL, _ id) {
	if h := cardTextFor(self); h != nil {
		h.updateFades()
	}
}

func cardClipBoundsChanged(self id, _ objc.SEL, _ id) {
	if h := cardTextFor(self); h != nil {
		h.updateFades()
	}
}

func (h *CardText) installFades() {
	if h == nil || h.scroll == 0 || h.fadeWatching {
		return
	}
	clip := send(h.scroll, "contentView")
	if clip == 0 {
		return
	}
	// The mask sits on the clip view, which is the text viewport. The
	// scroller is a sibling, and the card frame is drawn outside this view.
	send(clip, "setDrawsBackground:", 0)
	send(clip, "setWantsLayer:", 1)
	if h.delegate != 0 {
		send(clip, "setPostsBoundsChangedNotifications:", 1)
		send(send(class("NSNotificationCenter"), "defaultCenter"), "addObserver:selector:name:object:",
			uintptr(h.delegate),
			uintptr(sel("snipClipBoundsChanged:")),
			uintptr(nsString("NSViewBoundsDidChangeNotification")),
			uintptr(clip))
		h.fadeWatching = true
	}
	h.updateFades()
}

func (h *CardText) updateFades() {
	if h == nil || h.closing || h.scroll == 0 || h.view == 0 {
		return
	}
	clip := send(h.scroll, "contentView")
	if clip == 0 {
		return
	}
	send(clip, "setWantsLayer:", 1)
	host := send(clip, "layer")
	if host == 0 {
		return
	}
	bounds := msgRect(host, sel("bounds"))
	if bounds.size.width <= 0 || bounds.size.height <= 0 {
		bounds = msgRect(clip, sel("bounds"))
	}
	vis := msgRect(h.view, sel("documentVisibleRect"))
	doc := msgRect(h.view, sel("bounds"))
	top, bottom := ScrollFades(doc.size.height, vis.size.height, vis.origin.y, fadeEpsilon)
	band := FadeBand(bounds.size.height, FadeHeightPt)
	stops := MaskStops(bounds.size.height, band, top, bottom)
	if len(stops) == 0 {
		send(host, "setMask:", 0)
		return
	}
	if h.fadeMask == 0 {
		h.fadeMask = newFadeMask()
	}
	if h.fadeMask == 0 {
		return
	}
	// bounds origin tracks the scroll, so the mask stays on the viewport
	// through a scroll and a live resize.
	setViewFrame(h.fadeMask, sel("setFrame:"), bounds)
	applyMaskStops(h.fadeMask, stops)
	send(host, "setMask:", uintptr(h.fadeMask))
}

func newFadeMask() id {
	layer := send(send(class("CAGradientLayer"), "alloc"), "init")
	if layer == 0 {
		return 0
	}
	// Location 0 is the bottom edge, matching MaskStops.
	setCGPoint(layer, sel("setStartPoint:"), nsPoint{x: 0.5, y: 0})
	setCGPoint(layer, sel("setEndPoint:"), nsPoint{x: 0.5, y: 1})
	return layer
}

func applyMaskStops(layer id, stops []MaskStop) {
	if layer == 0 || colorSRGB == nil || numDouble == nil || len(stops) == 0 {
		return
	}
	colors := send(send(class("NSMutableArray"), "alloc"), "init")
	locs := send(send(class("NSMutableArray"), "alloc"), "init")
	for _, stop := range stops {
		c := colorSRGB(class("NSColor"), sel("colorWithSRGBRed:green:blue:alpha:"), 0, 0, 0, stop.Alpha)
		n := numDouble(class("NSNumber"), sel("numberWithDouble:"), stop.Location)
		if c == 0 || n == 0 {
			continue
		}
		cg := send(c, "CGColor")
		if cg == 0 {
			continue
		}
		send(colors, "addObject:", uintptr(cg))
		send(locs, "addObject:", uintptr(n))
	}
	send(layer, "setColors:", uintptr(colors))
	send(layer, "setLocations:", uintptr(locs))
	send(colors, "release")
	send(locs, "release")
}

func (h *CardText) removeFadeWatch() {
	if h == nil {
		return
	}
	if h.fadeWatching && h.delegate != 0 {
		send(send(class("NSNotificationCenter"), "defaultCenter"), "removeObserver:", uintptr(h.delegate))
		h.fadeWatching = false
	}
	if h.scroll != 0 {
		if clip := send(h.scroll, "contentView"); clip != 0 {
			if host := send(clip, "layer"); host != 0 {
				send(host, "setMask:", 0)
			}
		}
	}
	if h.fadeMask != 0 {
		send(h.fadeMask, "release")
		h.fadeMask = 0
	}
}

func snipScrollHitTest(self id, _ objc.SEL, pt nsPoint) id {
	h := cardTextFor(self)
	editing := h != nil && h.editing
	if !ScrollHit(editing, currentEventType(), pointInScroller(self, pt)) {
		return 0
	}
	if scrollHitImp == nil {
		return self
	}
	return scrollHitImp(self, sel("hitTest:"), pt)
}

func pointInScroller(scroll id, pt nsPoint) bool {
	scroller := send(scroll, "verticalScroller")
	if scroller == 0 {
		return false
	}
	r := msgRect(scroller, sel("frame"))
	return pointInNSRect(pt.x, pt.y, r.origin.x, r.origin.y, r.size.width, r.size.height)
}

func currentEventType() int {
	ev := currentEvent()
	if ev == 0 {
		return 0
	}
	return sendInt(ev, "type")
}

func currentEvent() id {
	return send(send(class("NSApplication"), "sharedApplication"), "currentEvent")
}

func commandDown() bool {
	ev := currentEvent()
	if ev == 0 {
		return false
	}
	return uintptr(send(ev, "modifierFlags"))&nsEventModifierCommand != 0
}

func cardDoCommand(self id, _ objc.SEL, _ id, cmd objc.SEL) bool {
	h := cardTextFor(self)
	if h == nil || !h.editing {
		return false
	}
	switch {
	case cmd == sel("cancelOperation:") || cmd == sel("cancel:"):
		if EditCommit(EditKeyEscape, false) {
			h.commit()
			return true
		}
		return false
	case cmd == sel("insertNewline:"):
		if EditCommit(EditKeyReturn, commandDown()) {
			h.commit()
			return true
		}
		return false
	default:
		return false
	}
}

func cardDidEndEditing(self id, _ objc.SEL, _ id) {
	if h := cardTextFor(self); h != nil {
		h.commit()
	}
}

func cardMenuForEvent(self id, _ objc.SEL, _ id, menu id, _ id, _ uintptr) id {
	h := cardTextFor(self)
	if h == nil {
		return menu
	}
	return autorelease(h.buildMenu())
}

func cardMenuAction(self id, _ objc.SEL, sender id) {
	h := cardTextFor(self)
	if h == nil || h.onAction == nil || sender == 0 {
		return
	}
	action := CardMenuAction(sendInt(sender, "tag"))
	switch action {
	case CardMenuEdit, CardMenuCopy, CardMenuDelete, CardMenuRestore:
		h.onAction(action)
	default:
		return
	}
}

func cardTextFor(key id) *CardText {
	cardTextMu.Lock()
	defer cardTextMu.Unlock()
	return cardTexts[key]
}

func goString(ns id) string {
	if ns == 0 {
		return ""
	}
	p := send(ns, "UTF8String")
	if p == 0 {
		return ""
	}
	n := sendInt(ns, "lengthOfBytesUsingEncoding:", nsUTF8StringEncoding)
	if n <= 0 {
		return ""
	}
	// Copy the UTF-8 bytes before the autorelease pool frees the NSString.
	// &p is the Go local that holds the pointer, which is the conversion
	// vet accepts for a pointer that arrived in an integer register.
	return string(unsafe.Slice(*(**byte)(unsafe.Pointer(&p)), n))
}
