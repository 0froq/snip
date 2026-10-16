package app

import "testing"

func TestPaperSlantStaysSmall(t *testing.T) {
	ids := []string{"", "a", "card-1", "hello world", "snip"}
	for _, id := range ids {
		s := paperSlant(id)
		if s < -0.7 || s > 0.7 {
			t.Fatalf("slant %v for %q", s, id)
		}
		again := paperSlant(id)
		if again != s {
			t.Fatalf("slant changed for %q", id)
		}
	}
}

func TestPaperCornersStayCut(t *testing.T) {
	tl, tr, br, bl := paperCorners("slip")
	for _, r := range []float32{tl, tr, br, bl} {
		if r < 1.5 || r > 3.5 {
			t.Fatalf("corner %v", r)
		}
	}
}
