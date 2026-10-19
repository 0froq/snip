package app

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"git.zacharyzhang.com/ZacharyZhang-NY/8bitMyGoUI/pixel"
	"github.com/0froq/snip/config"
	"github.com/0froq/snip/internal/macpaste"
	"github.com/0froq/snip/internal/macplace"
	"github.com/0froq/snip/secrets"
	"github.com/0froq/snip/store"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// App coordinates clipboard watching, cards, and shortcuts.
type App struct {
	cfg      config.Config
	store    *store.Store
	cards    map[string]*cardUI
	order    []string
	selected int
	// visible is the global hide/show state. Cards start shown. The flag is
	// not persisted.
	visible bool

	mu               sync.Mutex
	lastChange       int
	ignoreNextChange bool
	// heldText/heldCount is the pasteboard generation at the last delete.
	// The same changeCount must not open a new card for that text.
	heldText  string
	heldCount int

	sec secrets.Config

	cardPositions     map[string]cardPosition
	configPath        string
	pollReset         chan struct{}
	settingsWin       *mygo.Window
	settingsForm      *config.Form
	settingsApps      []macpaste.RunningApp
	settingsScroll    ui.ScrollState
	settingsScrollTab int
	rec               config.Recorder
	bindNotes         map[int]bindNote
	claimShortcut     func(string) error
	leaderOn          bool
	leaderGen         int
	leaderKeys        []string
	jumpLabels        map[string]string
	jumpPrefix        string
	twoKeyJumps       bool
	whichItems        []whichKeyItem
	whichWin          *mygo.Window
	taps              tapGate
	tray              *mygo.Tray
}

type cardPosition struct {
	x int
	y int
}

type cardUI struct {
	card      store.Card
	win       *mygo.Window
	display   string
	selected  bool
	pulse     bool
	hiding    bool
	hideGen   uint64
	userW     int
	userH     int
	userLines int
	rendered  string

	pointerActive  bool
	resizeEdge     int
	hoverEdge      int
	startX, startY int
	startW, startH int
	pointerDragged bool
	dragOffX       int
	dragOffY       int
	pressScreenX   int
	pressScreenY   int
	pressClicks    int
	closeHot       bool
	closePress     bool
	syncing        bool
	// resizeDepth is non-zero while a resize callback is on the stack.
	// hide/orderOut must not run from that stack.
	resizeDepth int
	jumpLabel   string
	badge       *mygo.Window
	textHost    *macplace.CardText
	closingEdit bool
}

// New creates the application state.
func New(cfg config.Config, configPath string) *App {
	return &App{
		cfg:        cfg,
		configPath: configPath,
		store:      store.New(),
		cards:      make(map[string]*cardUI),
		visible:    true,
		sec: secrets.Config{
			Enabled:                  cfg.Secrets.MaskEnabled,
			ExtraPatterns:            cfg.Secrets.ExtraPatterns,
			MinEntropyBits:           cfg.Secrets.MinEntropyBits,
			MinSecretLength:          cfg.Secrets.MinSecretLength,
			HonorPasteboardConcealed: cfg.Pasteboard.HonorConcealed,
		},
		cardPositions: make(map[string]cardPosition),
		pollReset:     make(chan struct{}, 1),
	}
}

// WhenReady registers UI hooks (call inside mygo.App.WhenReady).
func (a *App) WhenReady() {
	mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
	macpaste.WatchFrontmost()
	a.installMenus()
	a.registerShortcuts()
	a.checkAccessibility()
	go a.watchClipboard()
}

func (a *App) checkAccessibility() {
	if !macpaste.Supported {
		return
	}
	if !macpaste.AccessibilityTrusted(false) {
		log.Println("snip: enable Accessibility for snip in System Settings (required for Cmd+V paste simulation)")
		macpaste.AccessibilityTrusted(true)
	}
}

func (a *App) config() config.Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg
}

func (a *App) watchClipboard() {
	if !macpaste.Supported {
		log.Println("snip: clipboard watch requires macOS (darwin/arm64 build)")
		return
	}
	a.lastChange = macpaste.ChangeCount()
	tick := time.NewTicker(a.config().PollInterval())
	defer tick.Stop()
	for {
		select {
		case <-a.pollReset:
			tick.Reset(a.config().PollInterval())
			continue
		case <-tick.C:
		}
		count := macpaste.ChangeCount()
		if count == a.lastChange {
			continue
		}
		a.mu.Lock()
		if a.ignoreNextChange {
			a.ignoreNextChange = false
			a.lastChange = count
			a.mu.Unlock()
			continue
		}
		a.mu.Unlock()

		text := strings.TrimSpace(macpaste.ReadPlainText())
		if text == "" {
			text = strings.TrimSpace(mygo.Clipboard.ReadText())
		}
		cfg := a.config()
		markers := macpaste.ReadMarkers()
		if cfg.Pasteboard.SkipClipboardChange(markers) {
			a.lastChange = count
			continue
		}
		a.lastChange = count
		if text == "" || a.pasteboardHeld(count, text) {
			continue
		}
		// The poll sees the change after it lands. The frontmost app then
		// is the source. A filtered copy is not stored and is not replayed.
		if !cfg.Capture.Accepts(macpaste.FrontBundleID()) {
			continue
		}
		concealed := cfg.Pasteboard.ConcealedOnCard(markers)
		if existing, ok := a.store.FindText(text); ok {
			mygo.RunOnMain(func() { a.surfaceCard(existing.ID) })
			continue
		}
		card, evicted := a.store.AddWithLimit(text, cfg.EffectiveMaxCards(), concealed)
		display := a.cardDisplay(card)
		cardCfg := cfg.Card
		width := cardCfg.EstimateWidth(display)
		height := cardCfg.EffectiveHeight()
		mode := cfg.EffectivePlacement()
		macplace.ResolveOrigin(mode, width, height, func(x, y int, ok bool) {
			mygo.RunOnMain(func() {
				for _, id := range evicted {
					a.detachCard(id)
				}
				a.spawnCard(card, x, y, ok)
			})
		})
	}
}

func (a *App) spawnCard(card store.Card, placeX, placeY int, havePlacement bool) {
	host := &cardUI{
		card:    card,
		display: a.cardDisplay(card),
	}
	a.mu.Lock()
	a.cards[card.ID] = host
	a.order = append(a.order, card.ID)
	a.selected = len(a.order) - 1
	a.mu.Unlock()

	idx := len(a.order) - 1
	cfg := a.config()
	cardCfg := cfg.Card
	layout := cardCfg.Layout(host.display, 0, 0)
	width, height := layout.Width, layout.Height
	host.rendered = layout.Text
	host.userW, host.userH = layout.Width, layout.Height
	x, y := a.cardOrigin(card.ID, idx, width, height, placeX, placeY, havePlacement)
	minH := cardCfg.OneLineHeight(width)

	win := mygo.NewWindow(mygo.WindowOptions{
		Title:     "snip",
		Width:     width,
		Height:    height,
		X:         x,
		Y:         y,
		MinWidth:  cardCfg.WidthFloor(host.display),
		MinHeight: minH,
		MaxWidth:  cardCfg.EffectiveMaxWidth(),
		MaxHeight: cardCfg.EffectiveMaxHeight(),
		// NewWindow otherwise calls Show(), making the note key before we
		// can configure it. Notes must only be shown via ShowInactive.
		Hidden:    true,
		Frameless: cfg.Window.EffectiveFrameless(),
		// AppKit needs the nonactivating panel style at creation time, even
		// when the legacy panel setting is disabled. Settings is unaffected.
		Panel:         macplace.Supported || cfg.Window.EffectivePanel(),
		AlwaysOnTop:   cfg.Window.AlwaysOnTop,
		DisableShadow: true,
		// Clear, so the frame's empty corner dots show the desktop.
		BackgroundColor: "transparent",
		Content:         ui.View(func(c *ui.Context) { a.cardView(c, host) }),
	})
	host.win = win
	win.SetMovable(true)
	win.OnMove(func() { a.rememberCardPosition(card.ID, win) })
	win.OnResize(func() {
		a.followLeaderBadge(host)
		// The frame that AppKit just committed is the line count. Skipping
		// this for the whole live resize left the previous multi-line string
		// painted into a one-line window when the last willResize ran after
		// inLiveResize had already gone false.
		a.relayoutFromWindow(host, macplace.InLiveResize(win.NativeHandle()))
	})
	macplace.UseFrameResizeCursors()
	macplace.PreventWindowFocus(win.NativeHandle())
	a.applyWindowFlags(win)
	minW, minH := a.cardMinSize(host.display)
	win.SetMinimumSize(minW, minH)
	// Style-mask changes drop the frame min. The content min is what stops a
	// borderless live resize from collapsing the card to zero height.
	macplace.SetContentMinSize(win.NativeHandle(), minW, minH)
	win.SetResizable(true)
	// SetResizable writes the style mask again and puts the content view
	// back at the title-bar inset. Square the window after that write.
	macplace.SquareWindow(win.NativeHandle())
	cleanupResize := macplace.ConstrainWindowResize(win.NativeHandle(), func(width, height int) (int, int) {
		host.beginResize()
		defer host.endResize()
		// width/height are the window frame (proposed, or the frame that
		// stuck when the live resize ends). Not a pointer delta.
		layout := a.commitCardFrame(host, width, height)
		win.Invalidate()
		return layout.Width, layout.Height
	})
	win.OnClosed(func() {
		cleanupResize()
		if host.textHost != nil {
			host.textHost.Close()
			host.textHost = nil
		}
		a.closeBadge(host)
	})
	a.attachCardText(host)
	a.syncCardHeight(host)
	// Hide-all stays in effect. The card is already stored and placed;
	// showCard orders it in only while cards are shown.
	a.showCard(host)
	a.refreshSelection()
	a.relayoutLeader()
}

func (a *App) applyWindowFlags(win *mygo.Window) {
	w := a.config().Window
	win.SetVisibleOnAllWorkspaces(w.VisibleOnAllWorkspaces)
	win.SetContentProtection(w.ContentProtection)
	win.SetAlwaysOnTop(w.AlwaysOnTop)
}

func (a *App) cardOrigin(id string, cascadeIndex int, cardW, cardH int, placeX, placeY int, havePlacement bool) (int, int) {
	a.mu.Lock()
	if p, ok := a.cardPositions[id]; ok {
		a.mu.Unlock()
		return p.x, p.y
	}
	a.mu.Unlock()
	var x, y int
	if havePlacement {
		x, y = placeX, placeY
	} else {
		c := a.config().Card
		x = c.CascadeOriginX + (cascadeIndex%8)*c.CascadeStepX
		y = c.CascadeOriginY + (cascadeIndex%12)*c.CascadeStepY
	}
	return macplace.AvoidCards(x, y, cardW, cardH, a.visibleCardRects(id), a.workAreaAt(x, y, cardW, cardH))
}

// visibleCardRects is the on-screen bounds of every card except id. Hidden
// cards are omitted: a new note may use the selection even when a dismissed
// card last sat there.
func (a *App) visibleCardRects(except string) []macplace.Rect {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.visible {
		return nil
	}
	out := make([]macplace.Rect, 0, len(a.cards))
	for id, h := range a.cards {
		if id == except || h == nil || h.win == nil {
			continue
		}
		w, ht := h.win.Size()
		if w < 1 || ht < 1 {
			continue
		}
		x, y := h.win.Position()
		out = append(out, macplace.Rect{X: x, Y: y, Width: w, Height: ht})
	}
	return out
}

func (a *App) workAreaAt(x, y, w, h int) macplace.Rect {
	areas := screenWorkAreas()
	if len(areas) == 0 {
		return macplace.Rect{}
	}
	displays := make([]macplace.Display, len(areas))
	for i, area := range areas {
		displays[i] = macplace.Display{WorkArea: area}
	}
	return macplace.WorkAreaForPoint(x+w/2, y+h/2, displays)
}

// fitCardWidth reflows at the card's current viewport. A viewport of 0 is
// the natural height, capped at max height. The string is never ellipsized.
func (a *App) fitCardWidth(host *cardUI, w int) config.CardLayout {
	w = a.clampCardWidth(host, w)
	cardCfg := a.config().Card
	var layout config.CardLayout
	if host.userH > 0 {
		layout = cardCfg.LayoutViewport(host.display, w, host.userH)
	} else {
		layout = cardCfg.Layout(host.display, w, 0)
	}
	return a.publishCardLayout(host, layout)
}

func (a *App) clampCardWidth(host *cardUI, w int) int {
	cardCfg := a.config().Card
	floor := cardCfg.WidthFloor(host.display)
	maxW := cardCfg.EffectiveMaxWidth()
	if w < floor {
		w = floor
	}
	if w > maxW {
		w = maxW
	}
	if w < 1 {
		w = 1
	}
	return w
}

// layoutAt measures a viewport without publishing it. A non-positive
// viewport is one line, which is what a collapsed drag proposes.
func (a *App) layoutAt(host *cardUI, w, viewport int) config.CardLayout {
	w = a.clampCardWidth(host, w)
	return a.config().Card.LayoutViewport(host.display, w, viewport)
}

// fitCardSize clamps a drag to one line, the measured text, and max height.
// A shorter viewport than the text scrolls; a taller one snaps back to the
// text so the card does not keep empty paper.
func (a *App) fitCardSize(host *cardUI, w, h int) config.CardLayout {
	return a.publishCardLayout(host, a.layoutAt(host, w, h))
}

func (a *App) publishCardLayout(host *cardUI, layout config.CardLayout) config.CardLayout {
	host.userLines = layout.FullLines
	host.rendered = layout.Text
	host.userW, host.userH = layout.Width, layout.Height
	return layout
}

// commitCardFrame publishes the line layout for a window frame. Callers pass
// the size the window actually has (or the size windowWillResize is about to
// apply), never a press-to-cursor delta. A non-positive frame snaps up to one
// line; it does not clear the text or hide the card.
func (a *App) commitCardFrame(host *cardUI, w, h int) config.CardLayout {
	if w < 1 || h < 1 {
		log.Printf("snip: resize frame %dx%d is below one line; snapping up", w, h)
	}
	return a.fitCardSize(host, w, h)
}

// relayoutFromWindow makes the drawn text match win's frame and, once the
// gesture has ended, snaps that frame to the whole-line size. During a live
// resize AppKit owns the origin; a mid-drag setFrame/setPosition was moving
// the card off the visible frame.
func (a *App) relayoutFromWindow(host *cardUI, live bool) {
	if host == nil || host.win == nil || host.syncing {
		return
	}
	host.beginResize()
	defer host.endResize()
	w, h := host.win.Size()
	layout := a.commitCardFrame(host, w, h)
	minW, minH := a.cardMinSize(host.display)
	below := w < minW || h < minH || w < 1 || h < 1
	a.syncCardText(host)
	if live && !below {
		host.win.Invalidate()
		return
	}
	if w != layout.Width || h != layout.Height {
		if below || w < 1 || h < 1 {
			log.Printf("snip: setSize path=resize-sync %dx%d -> %dx%d live=%v", w, h, layout.Width, layout.Height, live)
		}
		host.syncing = true
		host.win.SetSize(layout.Width, layout.Height)
		host.syncing = false
		// SetSize can be ignored while AppKit is tearing the live resize
		// down. Republish from the frame that remains so a rejected grow
		// cannot leave the old multi-line string in the short window.
		aw, ah := host.win.Size()
		if aw != layout.Width || ah != layout.Height {
			layout = a.commitCardFrame(host, aw, ah)
		}
	}
	if !live {
		a.keepCardOnScreen(host)
	}
	// SetSize above can settle on a different frame than the one synced
	// before it. The scroll view follows the frame that remains.
	a.syncCardText(host)
	host.win.Invalidate()
}

// cardMinSize is the smallest content box: the width floor, and the height of
// one line at that width. Both are at least 1 point.
func (a *App) cardMinSize(display string) (int, int) {
	cardCfg := a.config().Card
	minW := cardCfg.WidthFloor(display)
	if minW < 1 {
		minW = 1
	}
	minH := cardCfg.OneLineHeight(minW)
	if minH < 1 {
		minH = 1
	}
	return minW, minH
}

// keepCardOnScreen clamps a live-resize frame that has collapsed below one
// line or left every display. A frame that still meets a work area is not
// moved, so dragging an edge against the screen does not fight the pointer.
func (a *App) keepCardOnScreen(host *cardUI) {
	if host == nil || host.win == nil || host.syncing {
		return
	}
	host.syncing = true
	defer func() { host.syncing = false }()
	w, h := host.win.Size()
	minW, minH := a.cardMinSize(host.display)
	if w < minW || h < minH {
		layout := a.commitCardFrame(host, w, h)
		log.Printf("snip: setSize path=resize-clamp %dx%d -> %dx%d", w, h, layout.Width, layout.Height)
		host.win.SetSize(layout.Width, layout.Height)
		w, h = layout.Width, layout.Height
	}
	areas := screenWorkAreas()
	if len(areas) == 0 {
		return
	}
	x, y := host.win.Position()
	nx, ny := macplace.SlideIntoVisible(x, y, w, h, areas)
	if nx != x || ny != y {
		log.Printf("snip: setPosition path=resize-clamp (%d,%d) -> (%d,%d) size=%dx%d", x, y, nx, ny, w, h)
		host.win.SetPosition(nx, ny)
	}
}

func screenWorkAreas() []macplace.Rect {
	displays := mygo.Screen.Displays()
	areas := make([]macplace.Rect, 0, len(displays))
	for _, d := range displays {
		wa := d.WorkArea
		if wa.Width <= 0 || wa.Height <= 0 {
			continue
		}
		areas = append(areas, macplace.Rect{X: wa.X, Y: wa.Y, Width: wa.Width, Height: wa.Height})
	}
	return areas
}

func (a *App) syncCardHeight(host *cardUI) {
	// Same commit as a resize end: the current frame picks the line count.
	// fitCardWidth(userLines) republished the max-line string when a fast
	// drag had left userLines at 0 and AppKit then ignored the grow.
	a.relayoutFromWindow(host, false)
}

func (a *App) rememberCardPosition(id string, win *mygo.Window) {
	x, y := win.Position()
	a.mu.Lock()
	a.cardPositions[id] = cardPosition{x: x, y: y}
	h := a.cards[id]
	a.mu.Unlock()
	a.followLeaderBadge(h)
}

func (a *App) cardView(c *ui.Context, host *cardUI) {
	cfg := a.config()
	// Read appearance before Use replaces the theme.
	dark := c.Theme().Dark
	k := usePixel(c, cfg, dark)
	font := cfg.Card.UIFont(0)
	ink := k.C(pixel.Ink)
	cardCfg := cfg.Card
	// No color tween and no scale: a selection swap is a palette slot, and
	// a scale would blur the dots. Enter motion stays off for the same reason.
	shell := ui.Column(c).Fill().FocusRing(false)
	shell = cardPointerInput(a, host, host.win, shell).Cursor(edgeCursor(host.hoverEdge))
	cardHot := shell.Hovered()
	if host.textHost != nil {
		host.textHost.SetColor(ink.R, ink.G, ink.B)
	}
	active := host.selected || host.pulse
	editing := host.textHost != nil && host.textHost.Editing()
	shell.Children(func() {
		ui.Column(c).Fill().FocusRing(false).Draw(func(p *ui.Painter, r ui.Rect) {
			drawCardChrome(k.Canvas(p, r), active, editing)
		}).Children(func() {
			ui.Column(c).Fill().
				Padding(float32(cardCfg.TopInset()), float32(cardCfg.FrameInsetX()), float32(cardCfg.FrameInsetY()), float32(cardCfg.FrameInsetX())).
				Children(func() {
					// macOS draws the string in the NSTextView, in the
					// configured face so code and CJK stay sharp. Other
					// platforms scroll the same face in the toolkit.
					if macplace.NativeCardText {
						return
					}
					ui.Scroll(c).Fill().Children(func() {
						ui.Text(c, host.display).
							Font(font.Family).
							FontSize(font.Size).
							TextColor(ink)
					})
				})
		})
		if cardHot || host.closePress {
			cardCloseButton(c, host.closeHot)
		}
	})
}

// cardCloseButton is the pixel close control in the title strip. The x
// shows only while the pointer is on the button.
func cardCloseButton(c *ui.Context, hot bool) {
	k := pixel.K(c)
	ui.Column(c.Key("card-close")).
		Absolute().
		Left(closeButtonX).
		Top(closeButtonY).
		Size(closeButtonSize, closeButtonSize).
		FocusRing(false).
		Draw(func(p *ui.Painter, r ui.Rect) {
			g := k.Canvas(p, r)
			if g.W < 3 || g.H < 3 {
				return
			}
			g.Frame(0, 0, g.W, g.H, pixel.Raised, pixel.Bad)
			if !hot {
				return
			}
			g.Line(2, 2, g.W-3, g.H-3, pixel.AccentInk)
			g.Line(g.W-3, 2, 2, g.H-3, pixel.AccentInk)
		})
}

func (a *App) toggleVisible() {
	a.mu.Lock()
	show := !a.visible
	a.mu.Unlock()
	a.setAllVisible(show)
}

func (a *App) setAllVisible(show bool) {
	a.mu.Lock()
	a.visible = show
	ids := append([]string(nil), a.order...)
	a.mu.Unlock()
	mygo.RunOnMain(func() {
		for _, id := range ids {
			a.mu.Lock()
			h := a.cards[id]
			a.mu.Unlock()
			if h == nil || h.win == nil {
				continue
			}
			if show {
				a.showCard(h)
			} else {
				a.hideCard(h)
			}
		}
	})
}

func (h *cardUI) beginResize() {
	if h != nil {
		h.resizeDepth++
	}
}

func (h *cardUI) endResize() {
	if h != nil && h.resizeDepth > 0 {
		h.resizeDepth--
	}
}

// cardOrderOutAllowed is false while a resize callback is running. Those
// callbacks must not order the window out or drop its alpha.
func cardOrderOutAllowed(h *cardUI) bool {
	return h != nil && h.resizeDepth == 0
}

func (a *App) hideCard(h *cardUI) {
	if h == nil || h.win == nil {
		return
	}
	if !cardOrderOutAllowed(h) {
		log.Printf("snip: resize blocked orderOut/alpha0 card=%s", h.card.ID)
		return
	}
	h.hideGen++
	h.hiding = false
	// Drop the whole window before any repaint. Fading or clearing the text
	// first left the rounded background on screen with the note gone.
	h.win.SetOpacity(0)
	h.win.Hide()
	a.hideBadge(h)
}

func (a *App) surfaceCard(id string) {
	a.selectID(id)
	if !a.cardsShown() {
		return
	}
	a.mu.Lock()
	h := a.cards[id]
	a.mu.Unlock()
	if h == nil || h.win == nil {
		return
	}
	h.pulse = true
	a.showCard(h)
	time.AfterFunc(280*time.Millisecond, func() {
		mygo.RunOnMain(func() {
			h.pulse = false
			if h.win != nil {
				h.win.Invalidate()
			}
		})
	})
}

func (a *App) selectID(id string) {
	a.mu.Lock()
	for i, oid := range a.order {
		if oid == id {
			a.selected = i
			break
		}
	}
	a.mu.Unlock()
	a.refreshSelection()
}

func (a *App) selectNumber(n int) {
	a.mu.Lock()
	if n < 1 || n > len(a.order) {
		a.mu.Unlock()
		return
	}
	id := a.order[len(a.order)-n]
	a.mu.Unlock()
	a.selectID(id)
}

// bringAllToFront shows every card and orders it front. The old pin action
// also flashed a color and a scale pulse, then hid the cards again; that
// effect is gone.
func (a *App) bringAllToFront() {
	a.setAllVisible(true)
}

func (a *App) invalidateCards() {
	a.mu.Lock()
	ids := append([]string(nil), a.order...)
	a.mu.Unlock()
	for _, id := range ids {
		a.mu.Lock()
		h := a.cards[id]
		a.mu.Unlock()
		if h != nil && h.win != nil {
			h.win.Invalidate()
		}
	}
}

func (a *App) moveSelection(delta int) {
	a.mu.Lock()
	if len(a.order) == 0 {
		a.mu.Unlock()
		return
	}
	a.selected += delta
	if a.selected < 0 {
		a.selected = len(a.order) - 1
	}
	if a.selected >= len(a.order) {
		a.selected = 0
	}
	a.mu.Unlock()
	a.refreshSelection()
}

func (a *App) refreshSelection() {
	a.mu.Lock()
	sel := a.selected
	ids := append([]string(nil), a.order...)
	a.mu.Unlock()
	mygo.RunOnMain(func() {
		for i, id := range ids {
			a.mu.Lock()
			h := a.cards[id]
			a.mu.Unlock()
			if h == nil || h.win == nil {
				continue
			}
			h.selected = i == sel
			h.win.Invalidate()
		}
		if a.leaderActive() {
			a.positionWhichKey()
		}
	})
}

func (a *App) pasteSelected() {
	a.mu.Lock()
	if len(a.order) == 0 {
		a.mu.Unlock()
		return
	}
	id := a.order[a.selected]
	h := a.cards[id]
	a.mu.Unlock()
	if h == nil {
		return
	}
	a.pasteCard(h)
}

func (a *App) attachCardText(host *cardUI) {
	if host == nil || host.win == nil {
		return
	}
	cfg := a.config().Card
	w, h := host.win.Size()
	ink := appearancePalette(false).C(pixel.Ink)
	host.textHost = macplace.AttachCardText(host.win.NativeHandle(), macplace.CardTextOptions{
		Text:         host.display,
		FontFamily:   cfg.EffectiveFontFamily(),
		FontSize:     float64(cfg.EffectiveFontSize()),
		Red:          ink.R,
		Green:        ink.G,
		Blue:         ink.B,
		CanEdit:      a.canEditCard(host.card),
		Titles:       a.cardMenuTitles(),
		Width:        w,
		Height:       h,
		PadX:         cfg.EffectivePaddingX(),
		PadY:         cfg.EffectivePaddingY(),
		Border:       config.CardBorderPx(),
		ResizeMargin: nativeResizeMargin,
		TopExtra:     config.CardTitleStripPx,
		OnAction: func(action macplace.CardMenuAction) {
			a.cardMenuAction(host, action)
		},
		OnCommit: func(text string) { a.commitCardEdit(host, text) },
	})
}

func (a *App) syncCardText(host *cardUI) {
	if host == nil || host.textHost == nil || host.win == nil {
		return
	}
	cfg := a.config().Card
	w, h := host.win.Size()
	host.textHost.SetTitles(a.cardMenuTitles())
	host.textHost.SetCanEdit(a.canEditCard(host.card))
	host.textHost.SetFont(cfg.EffectiveFontFamily(), float64(cfg.EffectiveFontSize()))
	host.textHost.Layout(w, h, cfg.EffectivePaddingX(), cfg.EffectivePaddingY(), config.CardBorderPx(), nativeResizeMargin, config.CardTitleStripPx)
	if !host.textHost.Editing() {
		host.textHost.SetText(host.display)
	}
}

func (a *App) cardMenuTitles() macplace.CardMenuTitles {
	lang := a.config().Language
	return macplace.CardMenuTitles{
		Edit:    tr(lang, "menu_edit"),
		Copy:    tr(lang, "menu_copy"),
		Delete:  tr(lang, "menu_delete"),
		Restore: tr(lang, "menu_restore"),
	}
}

func (a *App) canEditCard(card store.Card) bool {
	return !secrets.IsSensitive(card.Text, a.secrets(), card.Concealed)
}

func (a *App) openCardMenu(host *cardUI, win *mygo.Window) {
	if host == nil || host.textHost == nil || win == nil {
		return
	}
	host.textHost.SetTitles(a.cardMenuTitles())
	host.textHost.SetCanEdit(a.canEditCard(host.card))
	w, h := win.Size()
	sp := mygo.Screen.CursorScreenPoint()
	wx, wy := win.Position()
	host.textHost.ShowMenu(sp.X-wx, sp.Y-wy, w, h)
}

func (a *App) cardMenuAction(host *cardUI, action macplace.CardMenuAction) {
	switch action {
	case macplace.CardMenuEdit:
		a.beginCardEdit(host)
	case macplace.CardMenuCopy:
		a.copyCard(host)
	case macplace.CardMenuDelete:
		a.deleteCard(host.card.ID)
	case macplace.CardMenuRestore:
		a.restoreCard(host)
	default:
		panic("snip: unhandled card menu action")
	}
}

func (a *App) beginCardEdit(host *cardUI) {
	if host == nil || host.textHost == nil || !a.canEditCard(host.card) {
		return
	}
	host.textHost.BeginEdit(host.card.Text)
}

func (a *App) commitCardEdit(host *cardUI, text string) {
	if host == nil || host.closingEdit {
		return
	}
	host.closingEdit = true
	defer func() { host.closingEdit = false }()
	updated, ok := a.store.UpdateText(host.card.ID, text)
	if !ok {
		return
	}
	host.card = updated
	host.display = a.cardDisplay(updated)
	a.reflowCard(host)
}

func (a *App) copyCard(host *cardUI) {
	if host == nil {
		return
	}
	text := host.card.Text
	if host.textHost != nil && host.textHost.Editing() {
		text = host.textHost.Text()
	}
	a.mu.Lock()
	a.ignoreNextChange = true
	a.mu.Unlock()
	mygo.Clipboard.WriteText(text)
}

func (a *App) restoreCard(host *cardUI) {
	if host == nil {
		return
	}
	if host.textHost != nil && host.textHost.Editing() {
		host.textHost.CancelEdit()
	}
	updated, ok := a.store.RestoreOriginal(host.card.ID)
	if !ok {
		return
	}
	host.card = updated
	host.display = a.cardDisplay(updated)
	a.reflowCard(host)
}

func (a *App) reflowCard(host *cardUI) {
	if host == nil || host.win == nil {
		return
	}
	w, _ := host.win.Size()
	layout := a.fitCardWidth(host, w)
	host.syncing = true
	host.win.SetSize(layout.Width, layout.Height)
	host.syncing = false
	a.syncCardText(host)
	host.win.Invalidate()
}

func (a *App) pasteCard(h *cardUI) {
	if h == nil {
		return
	}
	text := h.card.Text
	a.mu.Lock()
	a.ignoreNextChange = true
	a.mu.Unlock()
	mygo.Clipboard.WriteText(text)
	if err := macpaste.PasteToPrevious(); err != nil {
		if errors.Is(err, macpaste.ErrAccessibility) {
			macpaste.OpenAccessibilitySettings()
			return
		}
		log.Printf("snip: paste: %v", err)
	}
}

func (a *App) deleteSelected() {
	a.mu.Lock()
	if len(a.order) == 0 {
		a.mu.Unlock()
		return
	}
	id := a.order[a.selected]
	a.mu.Unlock()
	a.deleteCard(id)
}

// clearCards closes every card window. The system pasteboard is left as it is.
func (a *App) clearCards() {
	a.mu.Lock()
	ids := append([]string(nil), a.order...)
	a.mu.Unlock()
	if len(ids) == 0 {
		return
	}
	a.holdPasteboard()
	for _, id := range ids {
		a.removeCard(id)
	}
}

// deleteCard closes one card window and drops snip's in-memory record.
// It does not write the system pasteboard.
func (a *App) deleteCard(id string) {
	a.holdPasteboard()
	a.removeCard(id)
}

// holdPasteboard remembers the text currently on the pasteboard and pins the
// watcher to that changeCount. Closing a card must not look like a new copy.
func (a *App) holdPasteboard() {
	count := macpaste.ChangeCount()
	text := strings.TrimSpace(macpaste.ReadPlainText())
	a.mu.Lock()
	a.lastChange = count
	if text != "" {
		a.heldText = text
		a.heldCount = count
	}
	a.mu.Unlock()
}

func (a *App) pasteboardHeld(count int, text string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return pasteboardHeld(a.heldText, a.heldCount, count, text)
}

// pasteboardHeld reports that this poll is still the pasteboard generation
// that was current when a card was deleted.
func pasteboardHeld(heldText string, heldCount, count int, text string) bool {
	return heldText != "" && text == heldText && count == heldCount
}

func (a *App) removeCard(id string) {
	mygo.RunOnMain(func() { a.detachCard(id) })
}

// detachCard closes a card window and updates selection. Call on the main thread.
func (a *App) detachCard(id string) {
	a.mu.Lock()
	h := a.cards[id]
	delete(a.cards, id)
	removedIdx := -1
	for i, oid := range a.order {
		if oid == id {
			removedIdx = i
			a.order = append(a.order[:i], a.order[i+1:]...)
			break
		}
	}
	if removedIdx >= 0 && removedIdx < a.selected {
		a.selected--
	}
	if a.selected >= len(a.order) && len(a.order) > 0 {
		a.selected = len(a.order) - 1
	}
	if len(a.order) == 0 {
		a.selected = 0
	}
	a.mu.Unlock()
	a.store.Delete(id)
	delete(a.cardPositions, id)
	if h != nil && h.win != nil {
		a.closeBadge(h)
		h.win.Close()
	}
	a.refreshSelection()
	a.relayoutLeader()
}

func (a *App) secrets() secrets.Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.sec
}

func (a *App) cardDisplay(card store.Card) string {
	return secrets.DisplayText(card.Text, a.secrets(), card.Concealed)
}

func (a *App) applyConfig(next config.Config) {
	a.mu.Lock()
	prev := a.cfg
	a.cfg = next
	a.sec = secrets.Config{
		Enabled:                  next.Secrets.MaskEnabled,
		ExtraPatterns:            next.Secrets.ExtraPatterns,
		MinEntropyBits:           next.Secrets.MinEntropyBits,
		MinSecretLength:          next.Secrets.MinSecretLength,
		HonorPasteboardConcealed: next.Pasteboard.HonorConcealed,
	}
	a.mu.Unlock()

	if prev.PollInterval() != next.PollInterval() {
		select {
		case a.pollReset <- struct{}{}:
		default:
		}
	}
	a.registerShortcuts()
	a.installMenus()

	for _, id := range a.store.TrimTo(next.EffectiveMaxCards()) {
		a.detachCard(id)
	}
	a.mu.Lock()
	ids := append([]string(nil), a.order...)
	a.mu.Unlock()
	for _, id := range ids {
		a.mu.Lock()
		h := a.cards[id]
		a.mu.Unlock()
		if h == nil || h.win == nil {
			continue
		}
		h.display = a.cardDisplay(h.card)
		a.applyWindowFlags(h.win)
		minW, minH := a.cardMinSize(h.display)
		h.win.SetMinimumSize(minW, minH)
		macplace.SetContentMinSize(h.win.NativeHandle(), minW, minH)
		h.win.SetMaximumSize(next.Card.EffectiveMaxWidth(), next.Card.EffectiveMaxHeight())
		if h.userW == 0 {
			width, height := next.Card.EstimateSize(h.display)
			h.win.SetSize(width, height)
		}
		a.syncCardHeight(h)
		h.win.Invalidate()
	}
}

// LoadConfigFromDefaultPaths loads config.toml from cwd or the user config dir.
// The path is where Settings writes, even when no file exists yet.
// LoadConfigFromDefaultPaths loads config. override wins when set (flag or env), even if the file is still missing.
func LoadConfigFromDefaultPaths(override string) (config.Config, string) {
	if override != "" {
		c, err := config.Load(override)
		if err != nil {
			return config.Default(), override
		}
		return c, override
	}
	cfg := config.Default()
	fallback := "config.toml"
	candidates := []string{"config.toml"}
	if dir, err := os.UserConfigDir(); err == nil {
		fallback = filepath.Join(dir, "snip", "config.toml")
		candidates = append(candidates, fallback)
	}
	for _, p := range candidates {
		c, err := config.Load(p)
		if err == nil {
			return c, p
		}
	}
	return cfg, fallback
}
