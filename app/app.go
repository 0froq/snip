package app

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/0froq/snip/config"
	"github.com/0froq/snip/internal/macpaste"
	"github.com/0froq/snip/secrets"
	"github.com/0froq/snip/store"
)

// App coordinates clipboard watching, cards, and shortcuts.
type App struct {
	cfg     config.Config
	store   *store.Store
	cards   map[string]*cardUI
	order   []string
	selected int
	visible  bool

	mu            sync.Mutex
	lastChange    int
	ignoreNextChange bool

	sec   secrets.Config

	cardPositions map[string]cardPosition
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
func New(cfg config.Config) *App {
	return &App{
		cfg:   cfg,
		store: store.New(),
		cards: make(map[string]*cardUI),
		sec: secrets.Config{
			Enabled:         cfg.Secrets.MaskEnabled,
			ExtraPatterns:   cfg.Secrets.ExtraPatterns,
			MinEntropyBits:  cfg.Secrets.MinEntropyBits,
			MinSecretLength: cfg.Secrets.MinSecretLength,
		},
		cardPositions: make(map[string]cardPosition),
	}
}

// WhenReady registers UI hooks (call inside mygo.App.WhenReady).
func (a *App) WhenReady() {
	mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
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
	reg := func(acc string, fn func()) {
		if acc == "" {
			return
		}
		if err := mygo.GlobalShortcut.Register(acc, fn); err != nil {
			log.Printf("snip: shortcut %q: %v", acc, err)
		}
	}
	reg(a.cfg.Shortcuts.ShowAll, func() { a.setAllVisible(true) })
	reg(a.cfg.Shortcuts.HideAll, func() { a.setAllVisible(false) })
	reg(a.cfg.Shortcuts.Next, func() { a.moveSelection(1) })
	reg(a.cfg.Shortcuts.Prev, func() { a.moveSelection(-1) })
	reg(a.cfg.Shortcuts.Paste, func() { a.pasteSelected() })
	reg(a.cfg.Shortcuts.Delete, func() { a.deleteSelected() })
}

func (a *App) watchClipboard() {
	if !macpaste.Supported {
		log.Println("snip: clipboard watch requires macOS (darwin/arm64 build)")
		return
	}
	a.lastChange = macpaste.ChangeCount()
	tick := time.NewTicker(a.cfg.PollInterval())
	defer tick.Stop()
	for range tick.C {
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
		card, evicted := a.store.AddWithLimit(text, a.cfg.EffectiveMaxCards())
		mygo.RunOnMain(func() {
			for _, id := range evicted {
				a.detachCard(id)
			}
			a.spawnCard(card)
		})
	}
}

func (a *App) spawnCard(card store.Card) {
	host := &cardUI{
		card:    card,
		display: secrets.DisplayText(card.Text, a.sec),
	}
	a.mu.Lock()
	a.cards[card.ID] = host
	a.order = append(a.order, card.ID)
	if len(a.order) == 1 {
		a.selected = 0
	}
	a.mu.Unlock()

	idx := len(a.order) - 1
	cardCfg := a.cfg.Card
	width := cardCfg.EstimateWidth(host.display)
	x, y := a.cardOrigin(card.ID, idx)

	win := mygo.NewWindow(mygo.WindowOptions{
		Title:         "snip",
		Width:         width,
		Height:        cardCfg.EffectiveHeight(),
		X:             x,
		Y:             y,
		Frameless:     a.cfg.Window.Frameless,
		Panel:         a.cfg.Window.Panel,
		AlwaysOnTop:   a.cfg.Window.AlwaysOnTop,
		DisableResize: true,
		StateKey:      "card-" + card.ID,
		Content:       ui.View(func(c *ui.Context) { a.cardView(c, host) }),
	})
	host.win = win
	win.SetMovable(true)
	win.OnMove(func() { a.rememberCardPosition(card.ID, win) })
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
	if a.cfg.Window.VisibleOnAllWorkspaces {
		win.SetVisibleOnAllWorkspaces(true)
	}
	if a.cfg.Window.ContentProtection {
		win.SetContentProtection(true)
	}
}

func (a *App) cardOrigin(id string, cascadeIndex int) (int, int) {
	a.mu.Lock()
	if p, ok := a.cardPositions[id]; ok {
		a.mu.Unlock()
		return p.x, p.y
	}
	a.mu.Unlock()
	c := a.cfg.Card
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
	border := ui.Transparent
	if host.selected {
		border = t.Accent
	}
	cardCfg := a.cfg.Card
	padX := float32(cardCfg.EffectivePaddingX())
	padY := float32(cardCfg.EffectivePaddingY())
	ui.Column(c).Fill().
		Padding(padY, padX, padY, padX).
		Background(t.Surface).Radius(8).
		Border(1, border).
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

// LoadConfigFromDefaultPaths loads config.toml from cwd or user config dir.
func LoadConfigFromDefaultPaths() config.Config {
	cfg := config.Default()
	candidates := []string{"config.toml"}
	if dir, err := os.UserConfigDir(); err == nil {
		candidates = append(candidates, filepath.Join(dir, "snip", "config.toml"))
	}
	for _, p := range candidates {
		c, err := config.Load(p)
		if err == nil {
			return c
		}
	}
	return cfg
}
