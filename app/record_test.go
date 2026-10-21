package app

import (
	"errors"
	"testing"
	"time"

	"git.zacharyzhang.com/ZacharyZhang-NY/8bitMyGoUI/pixel"
	"github.com/0froq/snip/config"
	"github.com/egoist/mygo/ui"
)

func TestRecordButtonCapturesChordOnKeyUp(t *testing.T) {
	a := New(config.Default(), "")
	form := config.FormFrom(a.config())
	form.SettingsTab = int(settingsShortcuts)
	a.settingsForm = &form
	tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c) }, 640, 900)
	if !tt.HasText("Record") {
		t.Fatal("missing record button")
	}
	if err := tt.Click("Record leader 0"); err != nil {
		t.Fatal(err)
	}
	if !a.rec.On || a.rec.Row != 0 || form.Bindings[0].Action != "leader" {
		t.Fatalf("record %+v action %s", a.rec, form.Bindings[0].Action)
	}
	prev := form.Bindings[0].Keys
	tt.KeyDown(ui.KeyK)
	if form.Bindings[0].Keys != prev {
		t.Fatalf("key down wrote %q", form.Bindings[0].Keys)
	}
	tt.KeyUp(ui.KeyK)
	if a.rec.On || form.Bindings[0].Keys != "K" {
		t.Fatalf("key up %+v keys %q", a.rec, form.Bindings[0].Keys)
	}
}

func TestRecordLeaderFollowUpAndTimeout(t *testing.T) {
	a := New(config.Default(), "")
	form := config.FormFrom(a.config())
	form.SettingsTab = int(settingsShortcuts)
	a.settingsForm = &form
	idx := bindingIndex(form.Bindings, "leader v")
	a.beginRecord(idx)
	tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c) }, 640, 900)
	tt.Key(ui.Cmd|ui.Shift, ui.KeySpace)
	if a.rec.Phase != config.RecWait || form.Bindings[idx].Keys != config.LeaderWaitText {
		t.Fatalf("wait phase %v keys %q", a.rec.Phase, form.Bindings[idx].Keys)
	}
	tt.Key(ui.Shift, ui.KeyD)
	if a.rec.On || form.Bindings[idx].Keys != "leader Shift+D" {
		t.Fatalf("follow-up on %v keys %q", a.rec.On, form.Bindings[idx].Keys)
	}
	plain := mustMatch(t, form.Bindings, "d")
	mod := mustMatch(t, form.Bindings, "Shift+D")
	if plain.Action == mod.Action {
		t.Fatal("shift+d matched plain d")
	}

	a.beginRecord(idx)
	tt = ui.NewTester(func(c *ui.Context) { a.settingsView(c) }, 640, 900)
	tt.Key(ui.Cmd|ui.Shift, ui.KeySpace)
	tt.Advance(config.LeaderWait)
	if form.Bindings[idx].Keys != "leader" || a.rec.On {
		t.Fatalf("timeout keys %q on %v", form.Bindings[idx].Keys, a.rec.On)
	}
	if !tt.HasText("Leader needs a following key.") {
		t.Fatalf("missing follow-up error in %v", tt.Texts())
	}
}

func TestRecordEscRestoresAndBackspaceClears(t *testing.T) {
	a := New(config.Default(), "")
	form := config.FormFrom(a.config())
	form.SettingsTab = int(settingsShortcuts)
	a.settingsForm = &form
	tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c) }, 640, 900)
	prev := form.Bindings[0].Keys
	if err := tt.Click("Record leader 0"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyBackspace)
	if !a.rec.On || form.Bindings[0].Keys != "" {
		t.Fatalf("clear on %v keys %q", a.rec.On, form.Bindings[0].Keys)
	}
	tt.Key(0, ui.KeyEscape)
	if a.rec.On || form.Bindings[0].Keys != prev {
		t.Fatalf("esc on %v keys %q want %q", a.rec.On, form.Bindings[0].Keys, prev)
	}
}

func TestRecordShowsConflictAndOccupied(t *testing.T) {
	a := New(config.Default(), "")
	form := config.FormFrom(a.config())
	form.SettingsTab = int(settingsShortcuts)
	a.settingsForm = &form
	tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c) }, 640, 900)
	if err := tt.Click("Record leader 0"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd|ui.Shift, ui.KeyV)
	if form.Bindings[0].Keys != "CmdOrCtrl+Shift+V" {
		t.Fatalf("keys %q", form.Bindings[0].Keys)
	}
	if !tt.HasText("Already used.") {
		t.Fatalf("missing conflict in %v", tt.Texts())
	}

	a.claimShortcut = func(string) error { return errors.New("busy") }
	if err := tt.Click("Record leader 0"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd|ui.Shift, ui.KeyK)
	if !tt.HasText("Used by another app.") {
		t.Fatalf("missing occupied in %v", tt.Texts())
	}
	idx := bindingIndex(form.Bindings, "leader v")
	form.Bindings[0].Keys = "CmdOrCtrl+Shift+Space"
	a.bindNotes = nil
	a.claimShortcut = func(string) error {
		t.Fatal("leader follow-up was registered as a global hotkey")
		return nil
	}
	a.beginRecord(idx)
	tt.SetSize(640, 900)
	tt.Key(ui.Cmd|ui.Shift, ui.KeySpace)
	tt.Key(0, ui.KeyD)
	if form.Bindings[idx].Keys != "leader d" {
		t.Fatalf("leader keys %q", form.Bindings[idx].Keys)
	}
}

func TestRecordCopyHasGlyphs(t *testing.T) {
	for _, key := range []string{"bind_record", "bind_occupied", "bind_leader_follow", "bind_hint"} {
		for _, r := range tr("zh", key) {
			if r > unicodeMaxASCII && !pixel.Has(r) {
				t.Fatalf("%s missing glyph %q", key, string(r))
			}
		}
	}
	if !pixel.Has('…') {
		t.Fatal("leader wait ellipsis")
	}
}

const unicodeMaxASCII = 127

func bindingIndex(rows config.BindingList, keys string) int {
	for i, row := range rows {
		if row.Keys == keys {
			return i
		}
	}
	return -1
}

func mustMatch(t *testing.T, rows config.BindingList, chord string) config.Binding {
	t.Helper()
	row, ok := config.MatchLeader(rows, chord)
	if !ok {
		t.Fatalf("no match %s", chord)
	}
	return row
}

func TestRecordTimeoutUsesTesterClock(t *testing.T) {
	if config.LeaderWait != 2*time.Second {
		t.Fatalf("wait %s", config.LeaderWait)
	}
}
