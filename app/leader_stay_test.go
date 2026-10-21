package app

import (
	"testing"

	"github.com/0froq/snip/config"
)

func TestStayKeepsLeaderMode(t *testing.T) {
	ended := false
	finishLeaderAction(true, func() { ended = true })
	if ended {
		t.Fatal("stay left leader mode")
	}
	finishLeaderAction(false, func() { ended = true })
	if !ended {
		t.Fatal("a binding without stay left leader mode open")
	}

	var stayed bool
	for _, row := range config.Default().Bindings {
		if row.Action != "next" || !row.Stay {
			continue
		}
		parsed, err := config.ParseBinding(row.Keys)
		if err != nil || !parsed.Leader {
			t.Fatalf("next stay row %q %v", row.Keys, err)
		}
		stayed = true
	}
	if !stayed {
		t.Fatal("default next leader binding does not stay")
	}
}

func TestLeaderDispatchKeepsModifiers(t *testing.T) {
	rows := config.Default().Bindings
	act, stay := leaderDispatch(rows, "Shift+D", "cancel", false)
	if act != "clear_cards" || stay {
		t.Fatalf("shift+d → %s stay %v", act, stay)
	}
	act, stay = leaderDispatch(rows, "d", "cancel", false)
	if act != "delete" || stay {
		t.Fatalf("d → %s stay %v", act, stay)
	}
	act, stay = leaderDispatch(rows, "shift+d", "cancel", true)
	if act != "clear_cards" || stay {
		t.Fatalf("live match ignored modifiers: %s %v", act, stay)
	}
	act, stay = leaderDispatch(rows, "escape", "cancel", true)
	if act != "cancel" || !stay {
		t.Fatalf("escape fell into a follow-up: %s %v", act, stay)
	}
	act, _ = leaderDispatch(rows, "Ctrl+D", "cancel", false)
	if act != "cancel" {
		t.Fatalf("ctrl+d ran %s", act)
	}
}
