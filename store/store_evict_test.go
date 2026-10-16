package store

import "testing"

func TestAddWithLimitNoEvictionUnderCap(t *testing.T) {
	s := New()
	for i := 0; i < 5; i++ {
		_, evicted := s.AddWithLimit("x", 20, false)
		if len(evicted) != 0 {
			t.Fatalf("unexpected eviction at i=%d: %v", i, evicted)
		}
	}
	if s.Len() != 5 {
		t.Fatalf("len %d", s.Len())
	}
}

func TestAddWithLimitEvictsFIFO(t *testing.T) {
	s := New()
	ids := make([]string, 0, 3)
	for _, text := range []string{"a", "b", "c"} {
		c, ev := s.AddWithLimit(text, 3, false)
		if len(ev) != 0 {
			t.Fatalf("add %q evicted early: %v", text, ev)
		}
		ids = append(ids, c.ID)
	}
	if s.Len() != 3 {
		t.Fatalf("len %d", s.Len())
	}
	newest, evicted := s.AddWithLimit("d", 3, false)
	if len(evicted) != 1 || evicted[0] != ids[0] {
		t.Fatalf("evicted %v want [%s]", evicted, ids[0])
	}
	if s.Len() != 3 {
		t.Fatalf("after eviction len %d", s.Len())
	}
	all := s.All()
	if len(all) != 3 || all[0].Text != "b" || all[2].Text != "d" || all[2].ID != newest.ID {
		t.Fatalf("order %+v", all)
	}
	if _, ok := s.Get(ids[0]); ok {
		t.Fatal("oldest should be gone from store")
	}
}

func TestAddWithLimitEvictsMultipleWhenBatchWouldOverflow(t *testing.T) {
	s := New()
	var ids []string
	for i := 0; i < 5; i++ {
		c, _ := s.AddWithLimit("old", 0, false) // unlimited fill
		ids = append(ids, c.ID)
	}
	_, evicted := s.AddWithLimit("new", 2, false)
	if len(evicted) != 4 {
		t.Fatalf("evicted %d want 4", len(evicted))
	}
	for i, id := range ids[:4] {
		if evicted[i] != id {
			t.Fatalf("evicted[%d]=%s want %s", i, evicted[i], id)
		}
	}
	if s.Len() != 2 {
		t.Fatalf("len %d", s.Len())
	}
	all := s.All()
	if all[0].ID != ids[4] || all[1].Text != "new" {
		t.Fatalf("remaining %+v", all)
	}
}

func TestAddWithLimitZeroMeansUnlimited(t *testing.T) {
	s := New()
	for i := 0; i < 25; i++ {
		_, ev := s.AddWithLimit("x", 0, false)
		if len(ev) != 0 {
			t.Fatal("unlimited should not evict")
		}
	}
	if s.Len() != 25 {
		t.Fatalf("len %d", s.Len())
	}
}
