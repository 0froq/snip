package app

import (
	"testing"

	"github.com/0froq/snip/config"
)

func TestPasteboardHeldOnlyForTheDeletedGeneration(t *testing.T) {
	if pasteboardHeld("", 4, 4, "hello") {
		t.Fatal("an empty hold must not swallow a copy")
	}
	if !pasteboardHeld("hello", 4, 4, "hello") {
		t.Fatal("the pasteboard text at the delete's changeCount must not open a card")
	}
	if pasteboardHeld("hello", 4, 5, "hello") {
		t.Fatal("a later copy of the same text is a new card")
	}
	if pasteboardHeld("hello", 4, 4, "other") {
		t.Fatal("a different pasteboard text is not the deleted card")
	}
}

func TestClearCardsBinding(t *testing.T) {
	cfg := config.Default()
	if keys := firstKeys(cfg, "clear_cards", false); keys != "CmdOrCtrl+Shift+Option+Backspace" {
		t.Fatalf("direct %q", keys)
	}
	if keys := firstKeys(cfg, "clear_cards", true); keys != "leader Shift+D" {
		t.Fatalf("leader %q", keys)
	}
	found := false
	for _, row := range cfg.Bindings {
		if row.Action == "clear_cards" {
			found = true
		}
	}
	if !found {
		t.Fatal("clear_cards is missing from settings bindings")
	}
	if tr("en", "clear_cards") == "clear_cards" || tr("zh", "clear_cards") == "clear_cards" {
		t.Fatal("clear_cards needs an en and zh label")
	}
}
