package store

import "testing"

func TestUpdateTextRemembersOriginalAndRestore(t *testing.T) {
	s := New()
	card := s.Add("clipboard")
	same, ok := s.UpdateText(card.ID, "clipboard")
	if !ok || same.Original != "" || same.Text != "clipboard" {
		t.Fatalf("unchanged edit: %+v", same)
	}
	edited, ok := s.UpdateText(card.ID, "draft")
	if !ok || edited.Text != "draft" || edited.Original != "clipboard" {
		t.Fatalf("first edit: %+v", edited)
	}
	again, ok := s.UpdateText(card.ID, "draft two")
	if !ok || again.Text != "draft two" || again.Original != "clipboard" {
		t.Fatalf("second edit: %+v", again)
	}
	back, ok := s.RestoreOriginal(card.ID)
	if !ok || back.Text != "clipboard" || back.Original != "" {
		t.Fatalf("restore: %+v", back)
	}
	stored, ok := s.Get(card.ID)
	if !ok || stored.Text != "clipboard" || stored.Original != "" {
		t.Fatalf("stored: %+v", stored)
	}
	again, ok = s.RestoreOriginal(card.ID)
	if !ok || again.Text != "clipboard" || again.Original != "" {
		t.Fatalf("second restore: %+v", again)
	}
	if _, ok := s.UpdateText("missing", "x"); ok {
		t.Fatal("missing id updated")
	}
	if _, ok := s.RestoreOriginal("missing"); ok {
		t.Fatal("missing id restored")
	}
}
