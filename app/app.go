package app

import (
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
}

type cardPosition struct {
	x int
	y int
}

type cardUI struct {
	card     store.Card
	win      *mygo.Window
	display  string
	selected bool
}

// New creates the application state.
func New(cfg config.Config, configPath string) *App {
	return &App{
		cfg:        cfg,
		configPath: configPath,
		store:      store.New(),
		cards:      make(map[string]*cardUI),
		sec: secrets.Config{
			Enabled:         cfg.Secrets.MaskEnabled,
			ExtraPatterns:   cfg.Secrets.ExtraPatterns,
			MinEntropyBits:  cfg.Secrets.MinEntropyBits,
			MinSecretLength: cfg.Secrets.MinSecretLength,
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
	if !macpaste.AccessibilityTrusted(true) {
		log.Println("snip: enable Accessibility for snip in System Settings (required for Cmd+V paste simulation)")
	}
}

func (a *App) registerShortcuts() {
	mygo.GlobalShortcut.UnregisterAll()
	sc := a.config().Shortcuts
	reg := func(acc string, fn func()) {
		if acc == "" {
			return
		}
		if err := mygo.GlobalShortcut.Register(acc, fn); err != nil {
			log.Printf("snip: shortcut %q: %v", acc, err)
		}
	}
	reg(sc.ShowAll, func() { a.setAllVisible(true) })
	reg(sc.HideAll, func() { a.setAllVisible(false) })
	reg(sc.Next, func() { a.moveSelection(1) })
	reg(sc.Prev, func() { a.moveSelection(-1) })
	reg(sc.Paste, func() { a.pasteSelected() })
	reg(sc.Delete, func() { a.deleteSelected() })
	reg(sc.Settings, func() { a.showSettings() })
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
		a.lastChange = count
		if text == "" {
			continue
		}
		cfg := a.config()
		card, evicted := a.store.AddWithLimit(text, cfg.EffectiveMaxCards())
		display := secrets.DisplayText(card.Text, a.secrets())
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
		display: secrets.DisplayText(card.Text, a.secrets()),
	}
	a.mu.Lock()
	a.cards[card.ID] = host
	a.order = append(a.order, card.ID)
	a.selected = len(a.order) - 1
	a.mu.Unlock()

	idx := len(a.order) - 1
	cfg := a.config()
	cardCfg := cfg.Card
	width := cardCfg.EstimateWidth(host.display)
	height := cardCfg.EffectiveHeight()
	x, y := a.cardOrigin(card.ID, idx, width, height, placeX, placeY, havePlacement)

	win := mygo.NewWindow(mygo.WindowOptions{
		Title:         "snip",
		Width:         width,
		Height:        height,
		X:             x,
		Y:             y,
		Frameless:     cfg.Window.Frameless,
		Panel:         cfg.Window.Panel,
		AlwaysOnTop:   cfg.Window.AlwaysOnTop,
		DisableResize: true,
		StateKey:      "card-" + card.ID,
		Content:       ui.View(func(c *ui.Context) { a.cardView(c, host) }),
	})
	host.win = win
	win.SetMovable(true)
	win.OnMove(func() { a.rememberCardPosition(card.ID, win) })
	win.OnFocus(func() { win.ShowInactive() })
	a.applyWindowFlags(win)
	if a.visible {
		win.ShowInactive()
	} else {
		a.visible = true
		win.ShowInactive()
	}
	a.refreshSelection()
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

func (a *App) rememberCardPosition(id string, win *mygo.Window) {
	x, y := win.Position()
	a.mu.Lock()
	a.cardPositions[id] = cardPosition{x: x, y: y}
	a.mu.Unlock()
}

func (a *App) cardView(c *ui.Context, host *cardUI) {
	t := c.Theme()
	surface := t.Surface
	if host.selected {
		surface = t.Surface.Mix(t.Accent, 0.12)
	}
	cardCfg := a.config().Card
	padX := float32(cardCfg.EffectivePaddingX())
	padY := float32(cardCfg.EffectivePaddingY())
	ui.Column(c).Fill().
		Padding(padY, padX, padY, padX).
		Background(surface).Radius(8).
		Border(1, t.Border).
		FocusRing(false).
		DragWindow().
		Children(func() {
			ui.Text(c, host.display).
				FontSize(float32(cardCfg.EffectiveFontSize())).
				Ellipsis("…").
				MaxLines(cardCfg.EffectiveMaxLines())
		})
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
				h.win.ShowInactive()
			} else {
				h.win.Hide()
			}
		}
	})
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
	text := h.card.Text
	a.mu.Lock()
	a.ignoreNextChange = true
	a.mu.Unlock()
	mygo.Clipboard.WriteText(text)
	if err := macpaste.PostCommandV(); err != nil {
		log.Printf("snip: paste: %v", err)
		macpaste.OpenAccessibilitySettings()
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
		h.win.Close()
	}
	a.refreshSelection()
}

func (a *App) secrets() secrets.Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.sec
}

func (a *App) applyConfig(next config.Config) {
	a.mu.Lock()
	prev := a.cfg
	a.cfg = next
	a.sec = secrets.Config{
		Enabled:         next.Secrets.MaskEnabled,
		ExtraPatterns:   next.Secrets.ExtraPatterns,
		MinEntropyBits:  next.Secrets.MinEntropyBits,
		MinSecretLength: next.Secrets.MinSecretLength,
	}
	a.mu.Unlock()

	if prev.PollInterval() != next.PollInterval() {
		select {
		case a.pollReset <- struct{}{}:
		default:
		}
	}
	a.registerShortcuts()

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
		h.display = secrets.DisplayText(h.card.Text, a.secrets())
		a.applyWindowFlags(h.win)
		width := next.Card.EstimateWidth(h.display)
		h.win.SetSize(width, next.Card.EffectiveHeight())
		h.win.Invalidate()
	}
}

// LoadConfigFromDefaultPaths loads config.toml from cwd or the user config dir.
// The path is where Settings writes, even when no file exists yet.
func LoadConfigFromDefaultPaths() (config.Config, string) {
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
