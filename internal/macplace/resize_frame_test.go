package macplace

import "testing"

func TestSnapResizeFrameKeepsTopAndLeft(t *testing.T) {
	x, y, w, h := snapResizeFrame(100, 200, 320, 180, 300, 40)
	if x != 100 || y != 340 || w != 300 || h != 40 {
		t.Fatalf("snapped frame = (%v,%v,%v,%v), want (100,340,300,40)", x, y, w, h)
	}
	// Growing keeps the same top edge (originY + old height).
	x, y, w, h = snapResizeFrame(10.5, 8, 100, 50.5, 100, 80)
	if x != 10.5 || y != 8+50.5-80 || w != 100 || h != 80 {
		t.Fatalf("grown frame = (%v,%v,%v,%v)", x, y, w, h)
	}
}
