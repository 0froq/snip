package store

import "testing"

func TestStoreAddAllDelete(t *testing.T) {
	s := New()
	a := s.Add("one")
	b := s.Add("two")
	if s.Len() != 2 {
		t.Fatalf("len %d", s.Len())
	}
	all := s.All()
	if len(all) != 2 || all[0].Text != "one" || all[1].Text != "two" {
		t.Fatalf("order: %+v", all)
	}
	if !s.Delete(a.ID) {
		t.Fatal("delete a")
	}
	if s.Len() != 1 {
		t.Fatalf("after delete len %d", s.Len())
	}
	got, ok := s.Get(b.ID)
	if !ok || got.Text != "two" {
		t.Fatalf("get b: %+v ok=%v", got, ok)
	}
	if s.Delete("missing") {
		t.Fatal("deleted missing")
	}
}

func TestTrimTo(t *testing.T) {
	s := New()
	s.Add("a")
	s.Add("b")
	s.Add("c")
	evicted := s.TrimTo(1)
	if len(evicted) != 2 || s.Len() != 1 {
		t.Fatalf("evicted %v len %d", evicted, s.Len())
	}
	if s.All()[0].Text != "c" {
		t.Fatalf("kept %+v", s.All())
	}
}
