package app

import (
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
)

func (a *App) registerShortcuts() {
	mygo.GlobalShortcut.UnregisterAll()
	a.leaderKeys = nil
	cfg := a.config()
	if cfg.Shortcuts.Direct {
		a.bind(cfg.Shortcuts.EffectiveToggle(), "toggle")
		a.bind(cfg.Shortcuts.Next, "next")
		a.bind(cfg.Shortcuts.Prev, "prev")
		a.bind(cfg.Shortcuts.Paste, "paste")
		a.bind(cfg.Shortcuts.Delete, "delete")
		a.bind(cfg.Shortcuts.Pin, "pin")
		a.bind(cfg.Shortcuts.Settings, "settings")
	}
	a.bind(cfg.Leader.Shortcut, "leader")
}

func (a *App) bind(acc, action string) {
	acc = strings.TrimSpace(acc)
	if acc == "" {
		return
	}
	if err := mygo.GlobalShortcut.Register(acc, func() { a.runShortcut(action, acc) }); err != nil {
		log.Printf("snip: shortcut %q: %v", acc, err)
	}
}

func (a *App) runShortcut(action, acc string) {
	cfg := a.config()
	double := ""
	if mapped, ok := cfg.Leader.Double[strings.ToLower(acc)]; ok {
		double = mapped
	}
	a.taps.press(acc, action, double, cfg.Leader.DoubleGap(), a.perform)
}

func (a *App) perform(action string) {
	switch action {
	case "toggle":
		a.toggleVisible()
	case "next":
		a.moveSelection(1)
	case "prev":
		a.moveSelection(-1)
	case "paste":
		a.pasteSelected()
	case "delete":
		a.deleteSelected()
	case "pin":
		a.flashAll()
	case "settings":
		a.showSettings()
	case "leader":
		a.armLeader()
	default:
		if n, err := strconv.Atoi(action); err == nil {
			a.selectNumber(n)
		}
	}
}

func (a *App) armLeader() {
	cfg := a.config()
	a.mu.Lock()
	a.leaderGen++
	gen := a.leaderGen
	a.leaderOn = true
	a.mu.Unlock()
	a.registerLeaderKeys()
	a.invalidateCards()
	time.AfterFunc(cfg.Leader.Timeout(), func() {
		mygo.RunOnMain(func() { a.expireLeader(gen) })
	})
}

func (a *App) expireLeader(gen int) {
	a.mu.Lock()
	if gen != a.leaderGen || !a.leaderOn {
		a.mu.Unlock()
		return
	}
	a.leaderOn = false
	a.mu.Unlock()
	a.clearLeaderKeys()
	a.invalidateCards()
}

func (a *App) registerLeaderKeys() {
	a.clearLeaderKeys()
	cfg := a.config().Leader
	keys := []struct{ key, action string }{
		{cfg.Toggle, "toggle"},
		{cfg.Next, "next"},
		{cfg.Prev, "prev"},
		{cfg.Paste, "paste"},
		{cfg.Delete, "delete"},
		{cfg.Pin, "pin"},
		{cfg.Settings, "settings"},
		{"escape", "cancel"},
	}
	for i := 1; i <= 9; i++ {
		keys = append(keys, struct{ key, action string }{strconv.Itoa(i), strconv.Itoa(i)})
	}
	for _, k := range keys {
		key := strings.TrimSpace(k.key)
		if key == "" {
			continue
		}
		action := k.action
		if err := mygo.GlobalShortcut.Register(key, func() {
			if action == "cancel" {
				a.endLeader()
				return
			}
			double := ""
			if mapped, ok := a.config().Leader.Double[strings.ToLower(key)]; ok {
				double = mapped
			}
			a.endLeader()
			a.taps.press(key, action, double, a.config().Leader.DoubleGap(), func(act string) {
				a.perform(act)
				a.invalidateCards()
			})
		}); err != nil {
			log.Printf("snip: leader key %q: %v", key, err)
			continue
		}
		a.leaderKeys = append(a.leaderKeys, key)
	}
}

func (a *App) clearLeaderKeys() {
	for _, k := range a.leaderKeys {
		mygo.GlobalShortcut.Unregister(k)
	}
	a.leaderKeys = nil
}

func (a *App) endLeader() {
	a.mu.Lock()
	a.leaderOn = false
	a.leaderGen++
	a.mu.Unlock()
	a.clearLeaderKeys()
	a.invalidateCards()
}

func (a *App) leaderActive() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.leaderOn
}

type tapGate struct {
	mu      sync.Mutex
	pending string
	token   int
}

func (g *tapGate) press(key, single, double string, gap time.Duration, fire func(string)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if double != "" && g.pending == key {
		g.pending = ""
		g.token++
		fire(double)
		return
	}
	if double == "" {
		fire(single)
		return
	}
	g.token++
	tok := g.token
	g.pending = key
	time.AfterFunc(gap, func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		if g.token == tok && g.pending == key {
			g.pending = ""
			fire(single)
		}
	})
}
