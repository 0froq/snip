package app

import (
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
)

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
	a.jumpPrefix = ""
	a.mu.Unlock()
	a.registerLeaderKeys()
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
	a.closeLeaderChrome()
}

func (a *App) registerLeaderKeys() {
	a.clearLeaderKeys()
	a.discardWhichKey()
	cfg := a.config()
	for _, item := range cfg.Bindings.NamedBindings() {
		a.bindLeader(item.Binding.Leader, item.Name, item.Binding.LeaderDouble, item.Binding.Sticky)
	}
	a.bindLeader("escape", "cancel", false, false)

	ids := a.jumpCardOrder()
	assigned := assignJumpLabels(ids, boundLeaderKeys(cfg.Bindings), jumpKeyPool)
	labels := make(map[string]string, len(ids))
	two := false
	for i, id := range ids {
		if i >= len(assigned) || assigned[i] == "" {
			continue
		}
		labels[assigned[i]] = id
		if len([]rune(assigned[i])) > 1 {
			two = true
		}
	}
	seen := map[string]struct{}{}
	for _, label := range assigned {
		for _, r := range label {
			k := string(r)
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			a.bindJump(k)
		}
	}
	a.mu.Lock()
	a.jumpLabels = labels
	a.jumpPrefix = ""
	a.twoKeyJumps = two
	a.mu.Unlock()
	a.whichItems = whichKeyItems(cfg)
	a.syncLeaderChrome()
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
	a.closeLeaderChrome()
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
