package app

import (
	"log"
	"strings"

	"github.com/0froq/snip/internal/macplace"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func (a *App) registerShortcuts() {
	mygo.GlobalShortcut.UnregisterAll()
	a.leaderKeys = nil
	a.mu.Lock()
	a.leaderOn = false
	a.leaderGen++
	a.mu.Unlock()
	a.closeLeaderChrome()
	cfg := a.config()
	for _, item := range cfg.Bindings.NamedBindings() {
		// Settings is the application menu's key equivalent. Carbon's
		// RegisterEventHotKey would consume it in every app, including
		// when snip is not the key app.
		if !registersGlobally(item.Name) {
			continue
		}
		a.bind(item.Binding.Direct, item.Name, item.Binding.DirectDouble)
	}
	a.bind(cfg.Leader.Shortcut, "leader", false)
}

func (a *App) relayoutLeader() {
	if !a.leaderActive() {
		return
	}
	a.registerLeaderKeys()
}

// jumpCardOrder is newest-first, matching the old digit shortcuts: 1 is the
// newest card.
func (a *App) jumpCardOrder() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	ids := make([]string, 0, len(a.order))
	for i := len(a.order) - 1; i >= 0; i-- {
		ids = append(ids, a.order[i])
	}
	return ids
}

func (a *App) bindLeader(key, action string, double, sticky bool) {
	key = strings.TrimSpace(key)
	if key == "" || a.leaderKeyTaken(key) {
		return
	}
	if err := mygo.GlobalShortcut.Register(key, func() {
		if action == "cancel" {
			a.endLeader()
			return
		}
		a.taps.pressOnly("leader:"+key, action, double, a.config().Leader.DoubleGap(), func(act string) {
			a.perform(act)
			a.invalidateCards()
			if !sticky {
				a.endLeader()
			}
		})
	}); err != nil {
		log.Printf("snip: leader key %q: %v", key, err)
		return
	}
	a.leaderKeys = append(a.leaderKeys, key)
}

func (a *App) bindJump(key string) {
	key = normalizeJumpKey(key)
	if key == "" || a.leaderKeyTaken(key) {
		return
	}
	if err := mygo.GlobalShortcut.Register(key, func() {
		mygo.RunOnMain(func() { a.handleJumpKey(key) })
	}); err != nil {
		log.Printf("snip: jump key %q: %v", key, err)
		return
	}
	a.leaderKeys = append(a.leaderKeys, key)
}

func (a *App) leaderKeyTaken(key string) bool {
	want := normalizeJumpKey(key)
	for _, k := range a.leaderKeys {
		if normalizeJumpKey(k) == want {
			return true
		}
	}
	return false
}

func (a *App) handleJumpKey(key string) {
	if !a.leaderActive() {
		return
	}
	key = normalizeJumpKey(key)
	a.mu.Lock()
	two := a.twoKeyJumps
	prefix := a.jumpPrefix
	var id string
	var matched bool
	if !two {
		id, matched = a.jumpLabels[key]
	} else if prefix == "" {
		matched = jumpHasPrefix(a.jumpLabels, key)
	} else {
		id, matched = a.jumpLabels[prefix+key]
	}
	a.mu.Unlock()
	if !two {
		if !matched {
			return
		}
		a.selectID(id)
		a.endLeader()
		return
	}
	if prefix == "" {
		if !matched {
			return
		}
		a.mu.Lock()
		a.jumpPrefix = key
		a.mu.Unlock()
		a.syncLeaderChrome()
		return
	}
	if matched {
		a.selectID(id)
		a.endLeader()
		return
	}
	a.mu.Lock()
	a.jumpPrefix = ""
	a.mu.Unlock()
	a.syncLeaderChrome()
}

func jumpHasPrefix(labels map[string]string, prefix string) bool {
	for label := range labels {
		if strings.HasPrefix(label, prefix) && len(label) > len(prefix) {
			return true
		}
	}
	return false
}

func (a *App) syncLeaderChrome() {
	if !a.leaderActive() {
		a.closeLeaderChrome()
		return
	}
	a.mu.Lock()
	ids := append([]string(nil), a.order...)
	a.mu.Unlock()
	seen := map[string]bool{}
	for _, id := range ids {
		seen[id] = true
		a.mu.Lock()
		h := a.cards[id]
		a.mu.Unlock()
		a.syncBadge(h)
	}
	a.mu.Lock()
	var extra []*cardUI
	for id, h := range a.cards {
		if !seen[id] {
			extra = append(extra, h)
		}
	}
	a.mu.Unlock()
	for _, h := range extra {
		a.hideBadge(h)
	}
	a.syncWhichKey()
	a.raiseLeaderChrome()
}

func (a *App) syncBadge(h *cardUI) {
	if h == nil || h.win == nil {
		return
	}
	text, ok := a.badgeText(h)
	if !ok || !h.win.IsVisible() {
		a.hideBadge(h)
		return
	}
	h.jumpLabel = text
	x, y := h.win.Position()
	w, ht := h.win.Size()
	f := BadgeFrame(frame{X: x, Y: y, W: w, H: ht}, BadgeDiameter)
	if h.badge == nil {
		host := h
		h.badge = a.newChromeWindow(f.X, f.Y, f.W, f.H, true, ui.View(func(c *ui.Context) {
			a.badgeView(c, host)
		}))
	} else {
		cx, cy := h.badge.Position()
		if cx != f.X || cy != f.Y {
			h.badge.SetPosition(f.X, f.Y)
		}
		bw, bh := h.badge.Size()
		if bw != f.W || bh != f.H {
			h.badge.SetSize(f.W, f.H)
		}
	}
	h.badge.Invalidate()
	if !h.badge.IsVisible() {
		showChrome(h.badge)
	}
}

func (a *App) badgeText(h *cardUI) (string, bool) {
	a.mu.Lock()
	labels := a.jumpLabels
	prefix := a.jumpPrefix
	a.mu.Unlock()
	full := ""
	for label, id := range labels {
		if id == h.card.ID {
			full = label
			break
		}
	}
	if full == "" {
		return "", false
	}
	if prefix == "" {
		return full, true
	}
	if strings.HasPrefix(full, prefix) && len(full) > len(prefix) {
		return full[len(prefix):], true
	}
	return "", false
}

func (a *App) hideBadge(h *cardUI) {
	if h == nil || h.badge == nil {
		return
	}
	h.badge.Hide()
}

func (a *App) closeBadge(h *cardUI) {
	if h == nil || h.badge == nil {
		return
	}
	w := h.badge
	h.badge = nil
	h.jumpLabel = ""
	w.Close()
}

func (a *App) followLeaderBadge(h *cardUI) {
	if h == nil || !a.leaderActive() {
		return
	}
	a.syncBadge(h)
	if a.whichWin != nil && a.selectedID() == h.card.ID {
		a.positionWhichKey()
	}
}

// raiseLeaderChrome keeps jump badges above card windows and the which-key
// window above the badges. Both stay non-activating.
func (a *App) raiseLeaderChrome() {
	if !a.leaderActive() {
		return
	}
	a.mu.Lock()
	ids := append([]string(nil), a.order...)
	a.mu.Unlock()
	for _, id := range ids {
		a.mu.Lock()
		h := a.cards[id]
		a.mu.Unlock()
		if h != nil && h.badge != nil && h.badge.IsVisible() {
			h.badge.ShowInactive()
		}
	}
	if a.whichWin != nil && a.whichWin.IsVisible() {
		a.whichWin.ShowInactive()
	}
}

func (a *App) selectedID() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.order) == 0 || a.selected < 0 || a.selected >= len(a.order) {
		return ""
	}
	return a.order[a.selected]
}

func (a *App) syncWhichKey() {
	if len(a.whichItems) == 0 {
		a.discardWhichKey()
		return
	}
	w, h := whichKeySize(a.whichItems, whichKeyFont(a.config().Card.EffectiveFontSize()))
	if a.whichWin == nil {
		a.whichWin = a.newChromeWindow(40, 40, w, h, false, ui.View(func(c *ui.Context) {
			a.whichKeyView(c)
		}))
	} else {
		cw, ch := a.whichWin.Size()
		if cw != w || ch != h {
			a.whichWin.SetMinimumSize(w, h)
			a.whichWin.SetMaximumSize(w, h)
			a.whichWin.SetSize(w, h)
		}
	}
	a.positionWhichKey()
}

func (a *App) discardWhichKey() {
	if a.whichWin == nil {
		return
	}
	w := a.whichWin
	a.whichWin = nil
	w.Close()
}

func (a *App) closeLeaderChrome() {
	a.mu.Lock()
	a.jumpLabels = nil
	a.jumpPrefix = ""
	a.twoKeyJumps = false
	a.whichItems = nil
	hosts := make([]*cardUI, 0, len(a.cards))
	for _, h := range a.cards {
		hosts = append(hosts, h)
	}
	a.mu.Unlock()
	for _, h := range hosts {
		a.closeBadge(h)
	}
	a.discardWhichKey()
}

// leaderAnchorPoint is the selected card's center when that card is showing,
// otherwise the mouse, so the which-key window sits on the display in use.
func (a *App) leaderAnchorPoint() (int, int) {
	a.mu.Lock()
	var h *cardUI
	if len(a.order) > 0 && a.selected >= 0 && a.selected < len(a.order) {
		h = a.cards[a.order[a.selected]]
	}
	a.mu.Unlock()
	if h != nil && h.win != nil && h.win.IsVisible() {
		x, y := h.win.Position()
		w, ht := h.win.Size()
		return x + w/2, y + ht/2
	}
	p := mygo.Screen.CursorScreenPoint()
	return p.X, p.Y
}

func (a *App) leaderWorkArea() frame {
	x, y := a.leaderAnchorPoint()
	d := mygo.Screen.DisplayNearestPoint(mygo.Point{X: x, Y: y})
	if area, ok := workFrame(d); ok {
		return area
	}
	if area, ok := workFrame(mygo.Screen.PrimaryDisplay()); ok {
		return area
	}
	return frame{}
}

func workFrame(d mygo.Display) (frame, bool) {
	wa := d.WorkArea
	if wa.Width > 0 && wa.Height > 0 {
		return frame{X: wa.X, Y: wa.Y, W: wa.Width, H: wa.Height}, true
	}
	b := d.Bounds
	if b.Width > 0 && b.Height > 0 {
		return frame{X: b.X, Y: b.Y, W: b.Width, H: b.Height}, true
	}
	return frame{}, false
}

func (a *App) positionWhichKey() {
	if a.whichWin == nil {
		return
	}
	screen := a.leaderWorkArea()
	if screen.W <= 0 || screen.H <= 0 {
		return
	}
	w, h := a.whichWin.Size()
	leader := a.config().Leader
	xPct, yPct := leader.WhichKeyPercents()
	x, y := placeWhichKey(screen, xPct, yPct, leader.EffectiveAlignX(), leader.EffectiveAlignY(), w, h, whichKeyMargin)
	cx, cy := a.whichWin.Position()
	if cx != x || cy != y {
		a.whichWin.SetPosition(x, y)
	}
	if !a.whichWin.IsVisible() {
		showChrome(a.whichWin)
	}
}

func (a *App) newChromeWindow(x, y, w, h int, transparent bool, content mygo.Content) *mygo.Window {
	placeX, placeY := x, y
	nudge := false
	if placeX == 0 && placeY == 0 {
		// NewWindow centers a window when both coordinates are zero.
		placeX = 1
		nudge = true
	}
	cfg := a.config()
	opts := mygo.WindowOptions{
		Title:             "snip",
		Width:             w,
		Height:            h,
		X:                 placeX,
		Y:                 placeY,
		MinWidth:          w,
		MinHeight:         h,
		MaxWidth:          w,
		MaxHeight:         h,
		Hidden:            true,
		Frameless:         true,
		Panel:             macplace.Supported || cfg.Window.EffectivePanel(),
		AlwaysOnTop:       cfg.Window.AlwaysOnTop,
		DisableShadow:     true,
		DisableResize:     true,
		DisableMove:       true,
		DisableMinimize:   true,
		DisableMaximize:   true,
		DisableClose:      true,
		DisableFullScreen: true,
		Transparent:       transparent,
		Content:           content,
	}
	if !transparent {
		// which-key: blur behind a translucent paper tint. The content
		// stays partly open so the material shows through.
		opts.Transparent = true
		opts.Vibrancy = mygo.VibrancyPopover
	}
	win := mygo.NewWindow(opts)
	if nudge {
		win.SetPosition(x, y)
	}
	win.SetMovable(false)
	win.SetResizable(false)
	win.SetIgnoreMouseEvents(true)
	macplace.PreventWindowFocus(win.NativeHandle())
	macplace.SquareWindow(win.NativeHandle())
	a.applyWindowFlags(win)
	return win
}

func showChrome(win *mygo.Window) {
	if win == nil {
		return
	}
	win.ShowInactive()
	win.Invalidate()
	macplace.DisplayWindow(win.NativeHandle())
}

func (a *App) badgeView(c *ui.Context, h *cardUI) {
	dark := c.Theme().Dark
	cfg := a.config()
	accent := activeColor(cfg.Card.ActiveColor, dark)
	label := h.jumpLabel
	size := cfg.Card.EffectiveFontSize()
	if len([]rune(label)) > 1 && size > 2 {
		size -= 2
	}
	font := cfg.Card.UIFont(size)
	th := paperTheme(dark, accent, font)
	th.Background = ui.Transparent
	c.SetTheme(th)
	ui.Column(c).Fill().PassThrough().Draw(func(p *ui.Painter, r ui.Rect) {
		inset := ui.Rect{X: r.X + 1, Y: r.Y + 1, W: r.W - 2, H: r.H - 2}
		if inset.W < 2 || inset.H < 2 {
			return
		}
		radius := inset.W / 2
		if inset.H < inset.W {
			radius = inset.H / 2
		}
		p.Fill(inset, th.Surface, radius)
		p.Stroke(inset, accent, radius, 1)
		if label == "" {
			return
		}
		layout := ui.ShapeText(label, font, 0)
		tw, thh := layout.Size()
		nudge := badgeGlyphNudge(font.Metrics(), font.Size)
		x := r.X + (r.W-tw)/2
		y := r.Y + (r.H-thh)/2 + nudge
		p.TextLayout(layout, x, y, paperInk(dark))
	})
}

func (a *App) whichKeyView(c *ui.Context) {
	dark := c.Theme().Dark
	cfg := a.config()
	accent := activeColor(cfg.Card.ActiveColor, dark)
	font := whichKeyFont(cfg.Card.EffectiveFontSize())
	c.SetTheme(paperTheme(dark, accent, font))
	t := c.Theme()
	items := append([]whichKeyItem(nil), a.whichItems...)
	var keyW float32
	for _, item := range items {
		w, _ := ui.ShapeText(item.Key, font, 0).Size()
		if w > keyW {
			keyW = w
		}
	}
	// A paper tint over the vibrancy material. Opaque text stays readable
	// in both appearances; the window itself is not a solid fill.
	fill := t.Surface.Alpha(0.82)
	if dark {
		fill = t.Surface.Alpha(0.78)
	}
	ui.Column(c).Fill().Padding(10, 12).Gap(4).
		Background(fill).
		Radius(float32(macplace.WindowCornerRadius)).
		Border(1, paperStroke(dark)).
		Children(func() {
			for _, item := range items {
				key, name := item.Key, item.Name
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, key).Width(keyW).Font(font.Family).FontSize(font.Size).TextColor(t.Accent)
					ui.Text(c, name).Font(font.Family).FontSize(font.Size).TextColor(paperInk(dark))
				})
			}
		})
}
