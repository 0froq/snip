package app

import "testing"

func TestPaperCornersMatch(t *testing.T) {
	tl, tr, br, bl := paperCorners("slip")
	if tl != paperRadius || tr != paperRadius || br != paperRadius || bl != paperRadius {
		t.Fatalf("corners %v %v %v %v", tl, tr, br, bl)
	}
}
