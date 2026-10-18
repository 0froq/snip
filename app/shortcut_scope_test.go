package app

import (
	"testing"

	"github.com/0froq/snip/config"
)

func TestSettingsShortcutIsMenuLocal(t *testing.T) {
	cfg := config.Default()
	if settingsAccelerator(cfg) != "CmdOrCtrl+," {
		t.Fatalf("settings shortcut %q", settingsAccelerator(cfg))
	}
	if keys := firstKeys(cfg, "clear_cards", false); keys != "CmdOrCtrl+Shift+Option+Backspace" {
		t.Fatalf("clear cards direct %q", keys)
	}
	if keys := firstKeys(cfg, "clear_cards", true); keys != "leader Shift+D" {
		t.Fatalf("clear cards leader %q", keys)
	}
	if keys := firstKeys(cfg, "toggle", false); keys != "CmdOrCtrl+Shift+S" {
		t.Fatalf("toggle shortcut %q", keys)
	}
	if keys := firstKeys(cfg, "leader", false); keys == "" {
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
	for _, row := range cfg.Bindings {
		if !registersGlobally(row.Action) {
			continue
		}
		parsed, err := config.ParseBinding(row.Keys)
		if err != nil || parsed.Leader {
			continue
		}
		global = append(global, parsed.Chord)
	}
	for _, acc := range global {
		if acc == settingsAccelerator(cfg) {
			t.Fatalf("global set includes %s", acc)
		}
	}
}

func firstKeys(cfg config.Config, action string, leader bool) string {
	for _, row := range cfg.Bindings {
		if row.Action != action {
			continue
		}
		parsed, err := config.ParseBinding(row.Keys)
		if err != nil || parsed.Leader != leader {
			continue
		}
		return row.Keys
	}
	return ""
}
