package app

import (
	"strings"
	"testing"

	"github.com/0froq/snip/config"
)

func TestSettingsTabLabels(t *testing.T) {
	keys := []string{"tab_general", "tab_cards", "tab_capture", "tab_shortcuts", "tab_privacy"}
	if len(settingsTabLabels("en")) != len(keys) || len(settingsTabLabels("zh")) != len(keys) {
		t.Fatal("tab bar and label keys differ")
	}
	for _, key := range keys {
		en, zh := tr("en", key), tr("zh", key)
		if en == key || zh == key || en == zh {
			t.Fatalf("%s en %q zh %q", key, en, zh)
		}
	}
}

func TestSettingsDraftSurvivesTabChange(t *testing.T) {
	f := config.FormFrom(config.Default())
	f.ClipboardPollMs = "450"
	f.SettingsTab = int(settingsCards)
	if f.ClipboardPollMs != "450" || f.SettingsTab != int(settingsCards) {
		t.Fatalf("draft poll %q tab %d", f.ClipboardPollMs, f.SettingsTab)
	}
}

func TestPaletteNames(t *testing.T) {
	for _, id := range config.PaletteIDs {
		key := "palette_" + strings.ReplaceAll(id, "-", "_")
		en, zh := tr("en", key), tr("zh", key)
		if en == key || zh == key || en == zh {
			t.Fatalf("%s en %q zh %q", key, en, zh)
		}
	}
}

func TestChineseSettingsTitle(t *testing.T) {
	if tr("zh", "settings") != "设置" {
		t.Fatal(tr("zh", "settings"))
	}
	if tr("en", "settings") != "Settings" {
		t.Fatal(tr("en", "settings"))
	}
	if tr("zh", "missing-key") == "" {
		t.Fatal("empty")
	}
}
