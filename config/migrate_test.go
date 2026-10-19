package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateLegacyShortcuts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	body := `
[leader]
shortcut = "Cmd+Space"
timeout_ms = 900

[bindings.toggle]
direct = "CmdOrCtrl+Shift+S"
leader = "s"

[bindings.next]
direct = "CmdOrCtrl+Shift+]"
leader = "j"
sticky = true
leader_double = true

[bindings.paste]
direct = "cmd+shift+v"
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Leader.Shortcut != "" {
		t.Fatalf("shortcut field still set %q", got.Leader.Shortcut)
	}
	if got.Leader.TimeoutMs != 900 {
		t.Fatalf("timeout %d", got.Leader.TimeoutMs)
	}
	want := BindingList{
		{Action: "leader", Keys: "Cmd+Space"},
		{Action: "toggle", Keys: "CmdOrCtrl+Shift+S"},
		{Action: "toggle", Keys: "leader s"},
		{Action: "next", Keys: "CmdOrCtrl+Shift+]"},
		{Action: "next", Keys: "leader j", Stay: true},
		{Action: "paste", Keys: "cmd+shift+v"},
	}
	if len(got.Bindings) != len(want) {
		t.Fatalf("bindings %#v", got.Bindings)
	}
	for i := range want {
		if got.Bindings[i] != want[i] {
			t.Fatalf("row %d got %#v want %#v", i, got.Bindings[i], want[i])
		}
	}
}

func TestMigrateModifiedLeaderFollowUp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	body := `
[leader]
shortcut = "CmdOrCtrl+Shift+Space"

[bindings.delete]
direct = "CmdOrCtrl+Shift+Backspace"
leader = "d"

[bindings.clear_cards]
direct = "CmdOrCtrl+Shift+Option+Backspace"
leader = "Shift+D"
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	var plain, mod Binding
	for _, row := range got.Bindings {
		switch row.Keys {
		case "leader d":
			plain = row
		case "leader Shift+D":
			mod = row
		}
	}
	if plain.Action != "delete" || mod.Action != "clear_cards" {
		t.Fatalf("migrated %#v", got.Bindings)
	}
	rows, err := CanonicalBindings(got.Bindings)
	if err != nil {
		t.Fatal(err)
	}
	del, ok1 := MatchLeader(rows, "d")
	clear, ok2 := MatchLeader(rows, "Shift+D")
	if !ok1 || !ok2 || del.Action != "delete" || clear.Action != "clear_cards" || del.Action == clear.Action {
		t.Fatalf("match plain %#v mod %#v", del, clear)
	}
}

func TestSaveCanonicalShortcutList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	src := Default()
	src.Bindings = BindingList{
		{Action: "leader", Keys: "cmd+shift+space"},
		{Action: "paste", Keys: "leader v", Stay: true},
		{Action: "paste", Keys: "cmd+shift+v", Stay: true},
	}
	form := FormFrom(src)
	saved, err := form.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Bindings[1].Keys != "leader v" || !saved.Bindings[1].Stay {
		t.Fatalf("leader row %#v", saved.Bindings[1])
	}
	if saved.Bindings[2].Keys != "CmdOrCtrl+Shift+V" || saved.Bindings[2].Stay {
		t.Fatalf("global row dropped stay? %#v", saved.Bindings[2])
	}
	if err := Save(path, saved); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Bindings) != 3 || got.Bindings[0].Keys != "CmdOrCtrl+Shift+Space" || got.Bindings[1].Stay != true {
		t.Fatalf("reloaded %#v", got.Bindings)
	}
}

func TestParseRejectsShortcutConflict(t *testing.T) {
	form := FormFrom(Default())
	form.Bindings = append(form.Bindings, Binding{Action: "delete", Keys: "cmd+shift+v"})
	if _, err := form.Parse(Default()); err == nil {
		t.Fatal("expected conflict")
	}
}
