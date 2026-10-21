package app

import (
	"runtime"
	"time"

	"github.com/0froq/snip/config"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type bindNote struct {
	keys string
	code string
}

func (a *App) beginRecord(row int) {
	if a.rec.On && a.rec.Row == row {
		return
	}
	if a.rec.On {
		a.cancelRecording()
	}
	f := a.settingsForm
	if f == nil || row < 0 || row >= len(f.Bindings) {
		return
	}
	delete(a.bindNotes, row)
	a.rec.Start(row, f.Bindings[row].Action, f.Bindings[row].Keys)
	a.suspendShortcuts()
}

func (a *App) cancelRecording() {
	if !a.rec.On {
		return
	}
	a.rec.Esc()
	a.syncRecText()
	a.rec = config.Recorder{}
	a.restoreShortcuts()
}

func (a *App) commitRecording() {
	a.syncRecText()
	row, text, issue := a.rec.Row, a.rec.Text, a.rec.Issue
	a.rec = config.Recorder{}
	if issue != "" {
		a.putNote(row, text, issue)
	} else {
		a.noteIfOccupied(row, text)
	}
	a.restoreShortcuts()
}

func (a *App) pollRecording(now time.Time) {
	if a.rec.Poll(now) {
		a.commitRecording()
	}
}

func (a *App) syncRecText() {
	f := a.settingsForm
	if f == nil || a.rec.Row < 0 || a.rec.Row >= len(f.Bindings) {
		return
	}
	f.Bindings[a.rec.Row].Keys = a.rec.Text
}

func (a *App) putNote(row int, keys, code string) {
	if row < 0 {
		return
	}
	if a.bindNotes == nil {
		a.bindNotes = map[int]bindNote{}
	}
	a.bindNotes[row] = bindNote{keys: keys, code: code}
}

func (a *App) noteIfOccupied(row int, keys string) {
	f := a.settingsForm
	if f == nil || row < 0 || row >= len(f.Bindings) {
		return
	}
	if !config.KnownAction(f.Bindings[row].Action) || !registersGlobally(f.Bindings[row].Action) {
		return
	}
	parsed, err := config.ParseBinding(keys)
	if err != nil || parsed.Leader {
		return
	}
	claim := a.claimShortcut
	if claim == nil {
		claim = claimGlobal
	}
	if err := claim(parsed.Chord); err != nil {
		a.putNote(row, keys, "occupied")
	}
}

func claimGlobal(acc string) error {
	if !mygo.App.IsReady() {
		return nil
	}
	if err := mygo.GlobalShortcut.Register(acc, func() {}); err != nil {
		return err
	}
	mygo.GlobalShortcut.Unregister(acc)
	return nil
}

func (a *App) suspendShortcuts() {
	if mygo.App.IsReady() {
		mygo.GlobalShortcut.UnregisterAll()
	}
	a.leaderKeys = nil
	a.mu.Lock()
	a.leaderOn = false
	a.leaderGen++
	a.mu.Unlock()
	a.closeLeaderChrome()
}

func (a *App) restoreShortcuts() {
	if !mygo.App.IsReady() {
		return
	}
	a.registerShortcuts()
}

func (a *App) recordCatcher(c *ui.Context) {
	if !a.rec.On {
		return
	}
	if a.rec.Phase == config.RecWait {
		d := a.rec.WaitUntil.Sub(c.Now())
		if d < 0 {
			d = 0
		}
		c.After(d)
	}
	ui.Box(c).Size(1, 1).InputMethod(false).HandleInput(a.recordInput).Focus()
}

func (a *App) recordInput(ev ui.InputEvent) bool {
	if !a.rec.On {
		return false
	}
	switch ev.Kind {
	case ui.InputKeyDown:
		if ev.Key == ui.KeyW && ev.Mods == ui.Cmd {
			return false
		}
		if ev.Repeat {
			return true
		}
		if ev.Key == ui.KeyEscape {
			return true
		}
		if ev.Key == ui.KeyBackspace {
			a.rec.Backspace()
			a.syncRecText()
			return true
		}
		if name := recordKeyName(ev.Key); name != "" {
			a.rec.Down(recordMods(ev.Mods), name)
		}
		return true
	case ui.InputKeyUp:
		if ev.Key == ui.KeyW && ev.Mods == ui.Cmd {
			return false
		}
		name := recordKeyName(ev.Key)
		var leaders []string
		if a.settingsForm != nil {
			leaders = config.LeaderChords(a.settingsForm.Bindings)
		}
		if a.rec.Up(name, ev.Time, leaders) {
			a.commitRecording()
			return true
		}
		a.syncRecText()
		return true
	case ui.InputText, ui.InputCompose, ui.InputCommand:
		return true
	case ui.InputPointerDown, ui.InputPointerUp, ui.InputPointerMove, ui.InputScroll, ui.InputSelect:
		return false
	default:
		panic("snip: unknown input kind")
	}
}

func recordMods(m ui.Modifiers) config.RecMods {
	cmd := m&ui.Cmd != 0 || m&ui.Super != 0
	ctrl := m&ui.Ctrl != 0 && ui.Ctrl != ui.Cmd
	if runtime.GOOS == "darwin" {
		cmd = m&ui.Super != 0
		ctrl = m&ui.Ctrl != 0
	}
	return config.RecMods{Cmd: cmd, Ctrl: ctrl, Alt: m&ui.Alt != 0, Shift: m&ui.Shift != 0}
}

func recordKeyName(k ui.Key) string {
	switch {
	case k >= ui.KeyA && k <= ui.KeyZ:
		return string(rune('a' + k - ui.KeyA))
	case k >= ui.Key0 && k <= ui.Key9:
		return string(rune('0' + k - ui.Key0))
	case k >= ui.KeyF1 && k <= ui.KeyF12:
		return "F" + itoa(int(k-ui.KeyF1)+1)
	}
	switch k {
	case ui.KeyEnter:
		return "Enter"
	case ui.KeyTab:
		return "Tab"
	case ui.KeySpace:
		return "Space"
	case ui.KeyDelete:
		return "Delete"
	case ui.KeyInsert:
		return "Insert"
	case ui.KeyHome:
		return "Home"
	case ui.KeyEnd:
		return "End"
	case ui.KeyPageUp:
		return "PageUp"
	case ui.KeyPageDown:
		return "PageDown"
	case ui.KeyLeft:
		return "Left"
	case ui.KeyRight:
		return "Right"
	case ui.KeyUp:
		return "Up"
	case ui.KeyDown:
		return "Down"
	case ui.KeyMinus:
		return "-"
	case ui.KeyEqual:
		return "="
	case ui.KeyComma:
		return ","
	case ui.KeyPeriod:
		return "."
	case ui.KeySlash:
		return "/"
	case ui.KeySemicolon:
		return ";"
	case ui.KeyQuote:
		return "'"
	case ui.KeyBracketLeft:
		return "["
	case ui.KeyBracketRight:
		return "]"
	case ui.KeyBackslash:
		return "\\"
	case ui.KeyBackquote:
		return "`"
	default:
		return ""
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [4]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
