package config

import (
	"testing"
	"time"
)

func TestParseModifiedLeaderFollowUp(t *testing.T) {
	plain, err := ParseBinding("leader d")
	if err != nil || !plain.Leader || plain.Chord != "D" || plain.Format() != "leader d" {
		t.Fatalf("plain %+v %v", plain, err)
	}
	mod, err := ParseBinding("leader shift+d")
	if err != nil || !mod.Leader || mod.Chord != "Shift+D" || mod.Format() != "leader Shift+D" {
		t.Fatalf("modified %+v %v", mod, err)
	}
	if plain.Chord == mod.Chord || SameChord(plain.Chord, mod.Chord) {
		t.Fatal("shift+d collapsed into d")
	}
	again, err := ParseBinding("leader Shift+D")
	if err != nil || !SameChord(again.Chord, mod.Chord) {
		t.Fatalf("spelling %+v %v", again, err)
	}
	rows, err := CanonicalBindings(DefaultBindings())
	if err != nil {
		t.Fatal(err)
	}
	del, ok1 := MatchLeader(rows, "d")
	clear, ok2 := MatchLeader(rows, "Shift+D")
	if !ok1 || !ok2 || del.Action != "delete" || clear.Action != "clear_cards" {
		t.Fatalf("default match %#v %#v", del, clear)
	}
	if clear.Keys != "leader Shift+D" || del.Keys != "leader d" {
		t.Fatalf("default spelling %#v %#v", del, clear)
	}
	got := CheckBindings([]Binding{
		{Action: "delete", Keys: "leader d"},
		{Action: "clear_cards", Keys: "leader shift+d"},
		{Action: "paste", Keys: "leader Shift+D"},
	})
	want := []string{"", "conflict", "conflict"}
	for i, code := range want {
		gotCode := ""
		if got[i] != nil {
			gotCode = got[i].Code
		}
		if gotCode != code {
			t.Fatalf("row %d %q want %q", i, gotCode, code)
		}
	}
}

func TestMatchLeaderKeepsModifiers(t *testing.T) {
	rows := DefaultBindings()
	clear, ok := MatchLeader(rows, "shift+d")
	if !ok || clear.Action != "clear_cards" {
		t.Fatalf("shift+d → %#v %v", clear, ok)
	}
	del, ok := MatchLeader(rows, "D")
	if !ok || del.Action != "delete" {
		t.Fatalf("d → %#v %v", del, ok)
	}
	if _, ok := MatchLeader(rows, "ctrl+d"); ok {
		t.Fatal("ctrl+d matched a follow-up")
	}
	if _, ok := MatchLeader(rows, "Shift+D+X"); ok {
		t.Fatal("unknown chord matched")
	}
	if chords := LeaderChords(rows); len(chords) != 1 || chords[0] != "CmdOrCtrl+Shift+Space" {
		t.Fatalf("leader chords %v", chords)
	}
}

func TestRecorderChordAndLeaderWait(t *testing.T) {
	var r Recorder
	r.Start(3, "clear_cards", "leader d")
	if r.Text != "leader d" || !r.On {
		t.Fatalf("start %+v", r)
	}
	now := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	leader := []string{"CmdOrCtrl+Shift+Space"}
	r.Down(RecMods{Cmd: true, Shift: true}, "Space")
	if r.Up("Space", now, leader) {
		t.Fatal("leader chord finished the row")
	}
	if r.Phase != RecWait || r.Text != LeaderWaitText || !r.On {
		t.Fatalf("wait %+v", r)
	}
	if r.Poll(now.Add(LeaderWait - time.Millisecond)) {
		t.Fatal("early poll finished")
	}
	r.Down(RecMods{Shift: true}, "d")
	if !r.Up("d", now.Add(time.Second), leader) {
		t.Fatal("follow-up did not finish")
	}
	if r.On || r.Text != "leader Shift+D" || r.Issue != "" {
		t.Fatalf("follow-up %+v", r)
	}

	r.Start(3, "clear_cards", "leader d")
	r.Down(RecMods{Cmd: true, Shift: true}, "Space")
	r.Up("Space", now, leader)
	if !r.Poll(now.Add(LeaderWait)) {
		t.Fatal("timeout did not finish")
	}
	if r.On || r.Text != "leader" || r.Issue != "leader_follow" {
		t.Fatalf("timeout %+v", r)
	}

	r.Start(0, "leader", "CmdOrCtrl+Shift+Space")
	r.Down(RecMods{Cmd: true, Shift: true}, "Space")
	if !r.Up("Space", now, leader) || r.Text != "CmdOrCtrl+Shift+Space" {
		t.Fatalf("leader row %+v", r)
	}
}

func TestRecorderEscAndBackspace(t *testing.T) {
	var r Recorder
	now := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	r.Start(1, "paste", "cmd+shift+v")
	r.Down(RecMods{Cmd: true}, "v")
	r.Esc()
	if r.On || r.Text != "cmd+shift+v" {
		t.Fatalf("esc %+v", r)
	}
	r.Start(1, "paste", "cmd+shift+v")
	r.Backspace()
	if !r.On || r.Text != "" || r.Phase != RecCapture {
		t.Fatalf("clear %+v", r)
	}
	r.Esc()
	if r.Text != "cmd+shift+v" {
		t.Fatalf("esc after clear %+v", r)
	}

	r.Start(2, "delete", "leader d")
	r.Down(RecMods{Cmd: true, Shift: true}, "Space")
	r.Up("Space", now, []string{"CmdOrCtrl+Shift+Space"})
	r.Backspace()
	if r.Phase != RecCapture || r.Text != "" || !r.On {
		t.Fatalf("backspace left the wait %+v", r)
	}
	r.Down(RecMods{}, "k")
	if !r.Up("k", now, nil) || r.Text != "K" {
		t.Fatalf("chord after clear %+v", r)
	}
	// A key-up for a different key does not finish the one still held.
	r.Start(2, "delete", "leader d")
	r.Down(RecMods{Shift: true}, "d")
	if r.Up("k", now, nil) {
		t.Fatal("other key finished the chord")
	}
	if !r.Up("d", now, nil) || r.Text != "Shift+D" {
		t.Fatalf("held key %+v", r)
	}
}
