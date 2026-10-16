package app

import "testing"

func TestSnapLines(t *testing.T) {
	if n := snapLines(2, 5, 17, 1, 6); n != 2 {
		t.Fatalf("inside a line: %d", n)
	}
	if n := snapLines(2, 17, 17, 1, 6); n != 3 {
		t.Fatalf("down one: %d", n)
	}
	if n := snapLines(2, -17, 17, 1, 6); n != 1 {
		t.Fatalf("up one: %d", n)
	}
	if n := snapLines(2, 80, 17, 1, 4); n != 4 {
		t.Fatalf("stop at text: %d", n)
	}
	if n := snapLines(2, -80, 17, 1, 4); n != 1 {
		t.Fatalf("stop at one: %d", n)
	}
}
