package macpaste

import "testing"

func TestAccessibilityPromptOpensOnce(t *testing.T) {
	var n int
	var p axPrompt
	if !p.allow(func() { n++ }) {
		t.Fatal("first request should open settings")
	}
	if p.allow(func() { n++ }) {
		t.Fatal("a later request should not open settings again")
	}
	if n != 1 {
		t.Fatalf("opened %d times", n)
	}
}
