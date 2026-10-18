package config

import "testing"

func TestParseBinding(t *testing.T) {
	global, err := ParseBinding("cmd+shift+v")
	if err != nil || global.Leader || global.Chord != "CmdOrCtrl+Shift+V" {
		t.Fatalf("cmd+shift+v → %+v %v", global, err)
	}
	same, err := ParseBinding("CmdOrCtrl+Shift+V")
	if err != nil || same != global {
		t.Fatalf("canonical %+v %v", same, err)
	}
	opt, err := ParseBinding("CmdOrCtrl+Shift+Option+Backspace")
	if err != nil || opt.Leader || opt.Chord != "CmdOrCtrl+Alt+Shift+Backspace" {
		t.Fatalf("option %+v %v", opt, err)
	}
	lead, err := ParseBinding("leader d")
	if err != nil || !lead.Leader || lead.Chord != "D" {
		t.Fatalf("leader d → %+v %v", lead, err)
	}
	shift, err := ParseBinding("leader Shift+D")
	if err != nil || !shift.Leader || shift.Chord != "Shift+D" {
		t.Fatalf("leader shift → %+v %v", shift, err)
	}
	if _, err := ParseBinding("   "); issueCode(t, err) != "empty" {
		t.Fatal(err)
	}
	if _, err := ParseBinding("leader"); issueCode(t, err) != "empty" {
		t.Fatal(err)
	}
	if _, err := ParseBinding("cmd+shift+nope"); issueCode(t, err) != "unknown" {
		t.Fatal(err)
	}
	if _, err := ParseBinding("cmd+foo+v"); issueCode(t, err) != "unknown" {
		t.Fatal(err)
	}
	plus, err := ParseBinding("cmd++")
	if err != nil || plus.Chord != "CmdOrCtrl++" {
		t.Fatalf("plus %+v %v", plus, err)
	}
}

func TestBindingConflicts(t *testing.T) {
	rows := []Binding{
		{Action: "paste", Keys: "cmd+shift+v"},
		{Action: "delete", Keys: "CmdOrCtrl+Shift+V"},
		{Action: "next", Keys: "leader d"},
		{Action: "prev", Keys: "leader D"},
		{Action: "pin", Keys: "leader shift+p"},
		{Action: "toggle", Keys: ""},
		{Action: "settings", Keys: "cmd+nope"},
		{Action: "leader", Keys: "leader x"},
		{Action: "leader", Keys: "cmd+shift+space"},
		{Action: "clear_cards", Keys: "cmd+shift+space"},
	}
	got := CheckBindings(rows)
	want := []string{"conflict", "conflict", "conflict", "conflict", "", "empty", "unknown", "leader", "conflict", "conflict"}
	if len(got) != len(want) {
		t.Fatalf("len %d", len(got))
	}
	for i, code := range want {
		gotCode := ""
		if got[i] != nil {
			gotCode = got[i].Code
		}
		if gotCode != code {
			t.Fatalf("row %d %q want %q (%+v)", i, gotCode, code, got[i])
		}
	}
	if got[6].Token != "nope" {
		t.Fatalf("unknown token %q", got[6].Token)
	}
}

func issueCode(t *testing.T, err error) string {
	t.Helper()
	issue, ok := err.(*BindingIssue)
	if !ok || issue == nil {
		t.Fatalf("error %v", err)
	}
	return issue.Code
}
