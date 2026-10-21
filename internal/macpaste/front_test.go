package macpaste

import "testing"

func TestFrontHistoryKeepsTheAppBeforeSnip(t *testing.T) {
	var h frontHistory
	if _, ok := h.previous(); ok {
		t.Fatal("no app has been frontmost yet")
	}
	h.setSelf(100)
	h.note(0)
	h.note(100)
	if _, ok := h.previous(); ok {
		t.Fatal("snip's own activation must not become the paste target")
	}
	h.note(200)
	got, ok := h.previous()
	if !ok || got != 200 {
		t.Fatalf("previous = %d, %v", got, ok)
	}
	h.note(100)
	got, ok = h.previous()
	if !ok || got != 200 {
		t.Fatalf("after snip activates, previous = %d, %v", got, ok)
	}
	h.note(300)
	got, ok = h.previous()
	if !ok || got != 300 {
		t.Fatalf("a later app should replace the target, got %d, %v", got, ok)
	}
	h.setSelf(300)
	if _, ok := h.previous(); ok {
		t.Fatal("learning that the stored pid is snip must drop it")
	}
}
