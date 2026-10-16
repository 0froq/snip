package app

import "testing"

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
