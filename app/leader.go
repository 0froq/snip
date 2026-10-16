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
	for _, item := range cfg.Bindings.NamedBindings() {
		a.bind(item.Binding.Direct, item.Name, item.Binding.DirectDouble)
	}
	a.bind(cfg.Leader.Shortcut, "leader", false)
}

func (a *App) bind(acc, action string, double bool) {
	acc = strings.TrimSpace(acc)
	if acc == "" {
		return
	}
	if err := mygo.GlobalShortcut.Register(acc, func() {
		a.taps.pressOnly(acc, action, double, a.config().Leader.DoubleGap(), a.perform)
	}); err != nil {
		log.Printf("snip: shortcut %q: %v", acc, err)
	}
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
	cfg := a.config()
	type leaderKey struct {
		key, action string
		double      bool
		sticky      bool
	}
	keys := make([]leaderKey, 0, 17)
	for _, item := range cfg.Bindings.NamedBindings() {
		keys = append(keys, leaderKey{
			key: item.Binding.Leader, action: item.Name,
			double: item.Binding.LeaderDouble, sticky: item.Binding.Sticky,
		})
	}
	keys = append(keys, leaderKey{key: "escape", action: "cancel"})
	for i := 1; i <= 9; i++ {
		keys = append(keys, leaderKey{key: strconv.Itoa(i), action: strconv.Itoa(i)})
	}
	for _, k := range keys {
		key := strings.TrimSpace(k.key)
		if key == "" {
			continue
		}
		action := k.action
		double := k.double
		sticky := k.sticky
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

// pressOnly fires immediately for a single binding. A double binding fires
// only on its second press and does nothing when its gap expires.
func (g *tapGate) pressOnly(key, action string, double bool, gap time.Duration, fire func(string)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !double {
		g.pending = ""
		g.token++
		fire(action)
		return
	}
	if g.pending == key {
		g.pending = ""
		g.token++
		fire(action)
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
		}
	})
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
