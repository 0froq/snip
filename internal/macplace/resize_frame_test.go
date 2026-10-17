package macplace

import "testing"

func TestSnapResizeFrameKeepsTopAndLeft(t *testing.T) {
	visible := box{X: 0, Y: 0, W: 10000, H: 10000}
	x, y, w, h := snapResizeFrame(100, 200, 320, 180, 300, 40, 1, 1, visible)
	if x != 100 || y != 340 || w != 300 || h != 40 {
		t.Fatalf("snapped frame = (%v,%v,%v,%v), want (100,340,300,40)", x, y, w, h)
	}
	// Growing keeps the same top edge (originY + old height).
	x, y, w, h = snapResizeFrame(10.5, 8, 100, 50.5, 100, 80, 1, 1, visible)
	if x != 10.5 || y != 8+50.5-80 || w != 100 || h != 80 {
		t.Fatalf("grown frame = (%v,%v,%v,%v)", x, y, w, h)
	}
	// A frame that still crosses the screen is not pulled back in.
	x, y, w, h = snapResizeFrame(-20, 100, 200, 80, 200, 80, 1, 1, box{0, 0, 1440, 800})
	if x != -20 || y != 100 || w != 200 || h != 80 {
		t.Fatalf("partially off frame moved: (%v,%v,%v,%v)", x, y, w, h)
	}
}

func TestSnapResizeFrameFastShrinkAndDragPastTop(t *testing.T) {
	visible := box{X: 0, Y: 0, W: 1440, H: 800}
	for _, newH := range []int{0, -80} {
		x, y, w, h := snapResizeFrame(100, 500, 240, 160, 80, newH, 120, 36, visible)
		if h != 36 || w < 120 || w != 120 || x != 100 || y != 500+160-36 {
			t.Fatalf("shrink %d: (%v,%v,%v,%v), want (100,624,120,36)", newH, x, y, w, h)
		}
		if !intersects(x, y, w, h, visible) {
			t.Fatalf("shrink %d left the screen: (%v,%v,%v,%v)", newH, x, y, w, h)
		}
	}
	// Top edge dragged to y=900, above a visible frame whose top is y=800.
	// Keeping that top would park a one-line card entirely off-screen.
	x, y, w, h := snapResizeFrame(40, 860, 240, 40, 240, 10, 120, 36, visible)
	if w != 240 || h != 36 || x != 40 || y != 764 {
		t.Fatalf("past top = (%v,%v,%v,%v), want (40,764,240,36)", x, y, w, h)
	}
	if !intersects(x, y, w, h, visible) {
		t.Fatal("drag past the top does not meet the visible frame")
	}
	// No screen: still clamp the size, and keep the (possibly off) top.
	x, y, w, h = snapResizeFrame(40, 860, 240, 40, 240, -5, 120, 36, box{})
	if x != 40 || y != 900-36 || w != 240 || h != 36 {
		t.Fatalf("no screen = (%v,%v,%v,%v)", x, y, w, h)
	}
}

func TestSlideIntoVisiblePicksNearestArea(t *testing.T) {
	areas := []Rect{
		{X: 0, Y: 0, Width: 800, Height: 600},
		{X: 2000, Y: 0, Width: 800, Height: 600},
	}
	// Already on the first display, even if it hangs off the left.
	x, y := SlideIntoVisible(-10, 40, 200, 80, areas)
	if x != -10 || y != 40 {
		t.Fatalf("on-screen origin moved to (%d,%d)", x, y)
	}
	// Entirely above the right display. The left display is farther.
	x, y = SlideIntoVisible(2200, -100, 120, 40, areas)
	if x != 2200 || y != 0 {
		t.Fatalf("nearest slide = (%d,%d), want (2200,0)", x, y)
	}
	x, y = SlideIntoVisible(10, 10, 40, 40, nil)
	if x != 10 || y != 10 {
		t.Fatalf("empty areas moved origin to (%d,%d)", x, y)
	}
}
