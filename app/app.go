package app

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

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
	visible  bool

	mu               sync.Mutex
	lastChange       int
	ignoreNextChange bool

	sec secrets.Config

	cardPositions map[string]cardPosition
	configPath    string
	pollReset     chan struct{}
	settingsWin   *mygo.Window
	settingsForm  *config.Form
	leaderOn      bool
	leaderGen     int
	leaderKeys    []string
	jumpLabels    map[string]string
	jumpPrefix    string
	twoKeyJumps   bool
	whichItems    []whichKeyItem
	whichWin      *mygo.Window
	taps          tapGate
	tray          *mygo.Tray
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
	syncing        bool
	// resizeDepth is non-zero while a resize callback is on the stack.
	// hide/orderOut must not run from that stack.
	resizeDepth int
	jumpLabel   string
	badge       *mygo.Window
}

// New creates the application state.
func New(cfg config.Config, configPath string) *App {
	return &App{
		cfg:        cfg,
		configPath: configPath,
		store:      store.New(),
		cards:      make(map[string]*cardUI),
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
		if text == "" {
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
	x, y := a.cardOrigin(card.ID, idx, width, height, placeX, placeY, havePlacement)
	minH := cardCfg.Layout(host.display, width, 1).Height

	win := mygo.NewWindow(mygo.WindowOptions{
		Title:     "snip",
		Width:     width,
		Height:    height,
		X:         x,
		Y:         y,
		MinWidth:  cardCfg.WidthFloor(host.display),
		MinHeight: minH,
		MaxWidth:  cardCfg.EffectiveMaxWidth(),
		MaxHeight: 10000,
		// NewWindow otherwise calls Show(), making the note key before we
		// can configure it. Notes must only be shown via ShowInactive.
		Hidden:    true,
		Frameless: cfg.Window.EffectiveFrameless(),
		// AppKit needs the nonactivating panel style at creation time, even
		// when the legacy panel setting is disabled. Settings is unaffected.
		Panel:           macplace.Supported || cfg.Window.EffectivePanel(),
		AlwaysOnTop:     cfg.Window.AlwaysOnTop,
		DisableShadow:   true,
		BackgroundColor: "light-dark(#F3EADF, #312924)",
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
	macplace.SquareWindow(win.NativeHandle())
	minW, minH := a.cardMinSize(host.display)
	win.SetMinimumSize(minW, minH)
	// Style-mask changes drop the frame min. The content min is what stops a
	// borderless live resize from collapsing the card to zero height.
	macplace.SetContentMinSize(win.NativeHandle(), minW, minH)
	win.SetResizable(true)
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
		a.closeBadge(host)
	})
	a.visible = true
	a.syncCardHeight(host)
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

func (a *App) cardOrigin(id string, cascadeIndex int, _ int, _ int, placeX, placeY int, havePlacement bool) (int, int) {
	a.mu.Lock()
	if p, ok := a.cardPositions[id]; ok {
		a.mu.Unlock()
		return p.x, p.y
	}
	a.mu.Unlock()
	if havePlacement {
		return placeX, placeY
	}
	c := a.config().Card
	return c.CascadeOriginX + (cascadeIndex%8)*c.CascadeStepX,
		c.CascadeOriginY + (cascadeIndex%12)*c.CascadeStepY
}

func (a *App) slipLines(host *cardUI, width int) int {
	cardCfg := a.config().Card
	if width <= 0 {
		width = cardCfg.EstimateWidth(host.display)
	}
	return cardCfg.Layout(host.display, width, 0).VisibleLines
}

// fitCardWidth is shared by the native pre-resize delegate and programmatic
// resizing. Height is never user-controlled: measure the exact (possibly
// ellipsized) text at the clamped width, using the configured line/height caps.
func (a *App) fitCardWidth(host *cardUI, w int) config.CardLayout {
	layout := a.layoutAt(host, w, host.userLines)
	host.rendered = layout.Text
	host.userW, host.userH = layout.Width, layout.Height
	return layout
}

// layoutAt measures a card without publishing host.userLines or host.rendered.
// A zero line count means the configured max.
func (a *App) layoutAt(host *cardUI, w, lines int) config.CardLayout {
	cardCfg := a.config().Card
	floor := cardCfg.WidthFloor(host.display)
	maxW := cardCfg.EffectiveMaxWidth()
	if w < floor {
		w = floor
	}
	if w > maxW {
		w = maxW
	}
	return cardCfg.Layout(host.display, w, lines)
}

// fitCardSize handles a drag: width as fitCardWidth, and the dragged height
// picks how many lines show, from one line up to max_lines, so the bottom
// edge stops there and never exposes blank space. The chosen count is
// published once, at the end. Zero userLines means "up to max_lines"; writing
// that zero before the snap would make a nested sync lay the card back out
// at full height.
func (a *App) fitCardSize(host *cardUI, w, h int) config.CardLayout {
	cardCfg := a.config().Card
	base := a.layoutAt(host, w, 0)
	maxN := min(base.FullLines, cardCfg.EffectiveMaxLines())
	if maxN < 1 {
		maxN = 1
	}
	one := a.layoutAt(host, base.Width, 1)
	if one.Height < 1 {
		one.Height = 1
	}
	if one.Width < 1 {
		one.Width = 1
	}
	lines := 1
	snapped := one
	// Anything shorter than one line box stays at that box. A fast drag can
	// propose 0 or a negative height; growing past it would hide the card.
	if h >= one.Height {
		for n := 1; n <= maxN; n++ {
			l := a.layoutAt(host, base.Width, n)
			if l.Height < 1 {
				l.Height = 1
			}
			if l.VisibleLines < n || l.Height > h {
				break
			}
			lines = n
			snapped = l
		}
		if lines == cardCfg.EffectiveMaxLines() && base.VisibleLines == lines {
			lines = 0
			snapped = a.layoutAt(host, base.Width, 0)
			if snapped.Height < one.Height {
				snapped.Height = one.Height
			}
		}
	}
	if snapped.Width < one.Width {
		snapped.Width = one.Width
	}
	if snapped.Height < one.Height {
		snapped.Height = one.Height
	}
	host.userLines = lines
	host.rendered = snapped.Text
	host.userW, host.userH = snapped.Width, snapped.Height
	return snapped
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
	minH := cardCfg.Layout(display, minW, 1).Height
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
	dark := c.Theme().Dark
	accent := activeColor(cfg.Card.ActiveColor, dark)
	font := cfg.Card.UIFont(0)
	c.SetTheme(paperTheme(dark, accent, font))
	t := c.Theme()
	surface := t.Surface
	border := paperEdge(dark)
	if host.selected || host.pulse {
		surface = t.Surface.Mix(accent, 0.14)
		border = accent
	}
	cardCfg := cfg.Card
	padX := float32(cardCfg.EffectivePaddingX())
	padY := float32(cardCfg.EffectivePaddingY())
	motion := ui.ElementTransition{Duration: cfg.Motion.Duration(), Colors: true}
	if cfg.Motion.Enabled {
		motion.Enter = &ui.Motion{Opacity: 0, Scale: 0.96, Y: 8}
	}
	col := ui.Column(c).Fill().
		Padding(padY, padX, padY, padX).
		Background(surface).
		Radius(float32(macplace.WindowCornerRadius)).
		Border(1, border).
		FocusRing(false)
	if cfg.Motion.Enabled {
		col = col.Transition(motion)
	}
	col = cardPointerInput(a, host, host.win, col).Cursor(edgeCursor(host.hoverEdge))
	col.Children(func() {
		// Draw from the frame, not a previously published string. A fast
		// resize can paint between the frame change and the snap.
		text, lines := host.rendered, max(a.slipLines(host, host.userW), 1)
		if host.win != nil {
			w, h := host.win.Size()
			if w > 0 && h > 0 {
				layout := a.commitCardFrame(host, w, h)
				text = layout.Text
				lines = max(layout.VisibleLines, 1)
			}
		}
		ui.Text(c, text).
			Font(font.Family).
			FontSize(font.Size).
			TextColor(paperInk(dark)).
			MaxLines(lines)
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

func (a *App) pasteCard(h *cardUI) {
	if h == nil {
		return
	}
	text := h.card.Text
	a.mu.Lock()
	a.ignoreNextChange = true
	a.mu.Unlock()
	mygo.Clipboard.WriteText(text)
	if err := macpaste.PostCommandV(); err != nil {
		log.Printf("snip: paste: %v", err)
		if errors.Is(err, macpaste.ErrAccessibility) {
			macpaste.OpenAccessibilitySettings()
		}
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
	a.removeCard(id)
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
		h.win.SetMaximumSize(next.Card.EffectiveMaxWidth(), 10000)
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
