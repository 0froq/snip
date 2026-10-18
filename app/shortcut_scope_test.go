package app

import (
	"testing"

	"github.com/0froq/snip/config"
)

func TestSettingsShortcutIsMenuLocal(t *testing.T) {
	cfg := config.Default()
	if cfg.Bindings.Settings.Direct != "CmdOrCtrl+," {
		t.Fatalf("settings shortcut %q", cfg.Bindings.Settings.Direct)
	}
	if cfg.Bindings.ClearCards.Direct != "CmdOrCtrl+Shift+Option+Backspace" || cfg.Bindings.ClearCards.Leader != "Shift+D" {
		t.Fatalf("clear cards binding %+v", cfg.Bindings.ClearCards)
	}
	if cfg.Bindings.Toggle.Direct != "CmdOrCtrl+Shift+S" {
		t.Fatalf("toggle shortcut %q", cfg.Bindings.Toggle.Direct)
	}
	if cfg.Leader.Shortcut == "" {
		t.Fatal("leader shortcut is empty")
	}
	if registersGlobally("settings") {
		t.Fatal("settings is registered as a global hotkey")
	}
	for _, action := range []string{"toggle", "next", "prev", "paste", "delete", "clear_cards", "pin", "leader"} {
		if !registersGlobally(action) {
			t.Fatalf("%s is not a global hotkey", action)
		}
	}
	var global []string
	for _, item := range cfg.Bindings.NamedBindings() {
		if !registersGlobally(item.Name) {
			continue
		}
		global = append(global, item.Binding.Direct)
	}
	global = append(global, cfg.Leader.Shortcut)
	for _, acc := range global {
		if acc == cfg.Bindings.Settings.Direct {
			t.Fatalf("global set includes %s", acc)
		}
	}
}
