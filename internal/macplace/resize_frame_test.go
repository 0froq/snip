package macplace

import (
	"math"
	"testing"
)

func TestTopAndRightDragsRelayoutWithoutMovingTheOrigin(t *testing.T) {
	const x, y, w, h = 40.0, 80.0, 300.0, 160.0
	cases := []struct {
		name        string
		edge        windowEdge
		originMoves bool
	}{
		{name: "top", edge: windowEdgeTop, originMoves: false},
		{name: "right", edge: windowEdgeRight, originMoves: false},
		{name: "bottom", edge: windowEdgeBottom, originMoves: true},
		{name: "left", edge: windowEdgeLeft, originMoves: true},
	}
	for _, tc := range cases {
		nx, ny, nw, nh := dragWindowEdge(x, y, w, h, 36, tc.edge)
		if !contentSizeChanged(w, h, nw, nh) {
			t.Fatalf("%s did not change the size (%v,%v) -> (%v,%v)", tc.name, w, h, nw, nh)
		}
		moved := nx != x || ny != y
		if moved != tc.originMoves {
			t.Fatalf("%s origin moved=%v, want %v (%v,%v) -> (%v,%v)", tc.name, moved, tc.originMoves, x, y, nx, ny)
		}
	}
	// Dragging the window keeps the size. That is not a relayout.
	if contentSizeChanged(w, h, w, h) {
		t.Fatal("an origin-only move relayouts")
	}
	if contentSizeChanged(w, h, 0, h) || contentSizeChanged(w, h, w, -1) {
		t.Fatal("a failed size read relayouts")
	}
}

func TestSnapResizeFrameKeepsTopAndLeft(t *testing.T) {
	visible := box{X: 0, Y: 0, W: 10000, H: 10000}
	x, y, w, h, ok := snapResizeFrame(100, 200, 320, 180, 300, 40, 1, 1, visible)
	if !ok || x != 100 || y != 340 || w != 300 || h != 40 {
		t.Fatalf("snapped frame = (%v,%v,%v,%v) ok=%v, want (100,340,300,40)", x, y, w, h, ok)
	}
	// Growing keeps the same top edge (originY + old height).
	x, y, w, h, ok = snapResizeFrame(10.5, 8, 100, 50.5, 100, 80, 1, 1, visible)
	if !ok || x != 10.5 || y != 8+50.5-80 || w != 100 || h != 80 {
		t.Fatalf("grown frame = (%v,%v,%v,%v) ok=%v", x, y, w, h, ok)
	}
	// A frame that still crosses the screen is not pulled back in.
	x, y, w, h, ok = snapResizeFrame(-20, 100, 200, 80, 200, 80, 1, 1, box{0, 0, 1440, 800})
	if !ok || x != -20 || y != 100 || w != 200 || h != 80 {
		t.Fatalf("partially off frame moved: (%v,%v,%v,%v) ok=%v", x, y, w, h, ok)
	}
}

func TestSnapResizeFrameAnchorsAndRejectsBadReads(t *testing.T) {
	visible := box{X: 0, Y: 0, W: 1440, H: 800}
	// Horizontal drag: width changes, origin does not.
	x, y, w, h, ok := snapResizeFrame(180, 420, 320, 160, 480, 160, 120, 36, visible)
	if !ok || x != 180 || y != 420 || w != 480 || h != 160 {
		t.Fatalf("horizontal = (%v,%v,%v,%v) ok=%v, want origin (180,420)", x, y, w, h, ok)
	}
	// Vertical shrink to one line keeps the left and the top. 420+160-36 = 544.
	x, y, w, h, ok = snapResizeFrame(180, 420, 320, 160, 320, 36, 120, 36, visible)
	if !ok || x != 180 || y != 544 || w != 320 || h != 36 {
		t.Fatalf("vertical = (%v,%v,%v,%v) ok=%v, want (180,544,320,36)", x, y, w, h, ok)
	}
	if x == 0 && y == 0 {
		t.Fatal("vertical shrink moved the card to the bottom-left")
	}
	// A zero intermediate frame is not a window at the origin.
	x, y, w, h, ok = snapResizeFrame(0, 0, 0, 0, 320, 36, 120, 36, visible)
	if ok || x != 0 || y != 0 || w != 0 || h != 0 {
		t.Fatalf("zero frame applied as (%v,%v,%v,%v) ok=%v", x, y, w, h, ok)
	}
	// Non-finite read: do not invent (0,0) or the screen origin.
	x, y, w, h, ok = snapResizeFrame(math.NaN(), 420, 320, 160, 320, 36, 120, 36, visible)
	if ok {
		t.Fatalf("non-finite frame applied as (%v,%v,%v,%v)", x, y, w, h)
	}
	// Screen lookup failed. Size still snaps; origin stays on the top-left anchor.
	x, y, w, h, ok = snapResizeFrame(180, 420, 320, 160, 200, 10, 120, 36, box{})
	if !ok || x != 180 || y != 420+160-36 || w != 200 || h != 36 {
		t.Fatalf("no screen = (%v,%v,%v,%v) ok=%v", x, y, w, h, ok)
	}
	ix, iy := SlideIntoVisible(180, 420, 320, 36, []Rect{{}})
	if ix != 180 || iy != 420 {
		t.Fatalf("zero work area moved origin to (%d,%d)", ix, iy)
	}
}

func TestSnapResizeFrameFastShrinkAndDragPastTop(t *testing.T) {
	visible := box{X: 0, Y: 0, W: 1440, H: 800}
	for _, newH := range []int{0, -80} {
		x, y, w, h, ok := snapResizeFrame(100, 500, 240, 160, 80, newH, 120, 36, visible)
		if !ok || h != 36 || w != 120 || x != 100 || y != 500+160-36 {
			t.Fatalf("shrink %d: (%v,%v,%v,%v) ok=%v, want (100,624,120,36)", newH, x, y, w, h, ok)
		}
		if !intersects(x, y, w, h, visible) {
			t.Fatalf("shrink %d left the screen: (%v,%v,%v,%v)", newH, x, y, w, h)
		}
	}
	// Top edge dragged to y=900, above a visible frame whose top is y=800.
	// Keeping that top would park a one-line card entirely off-screen.
	x, y, w, h, ok := snapResizeFrame(40, 860, 240, 40, 240, 10, 120, 36, visible)
	if !ok || w != 240 || h != 36 || x != 40 || y != 764 {
		t.Fatalf("past top = (%v,%v,%v,%v) ok=%v, want (40,764,240,36)", x, y, w, h, ok)
	}
	if !intersects(x, y, w, h, visible) {
		t.Fatal("drag past the top does not meet the visible frame")
	}
	// No screen: still clamp the size, and keep the anchored top. Not (0,0).
	x, y, w, h, ok = snapResizeFrame(40, 860, 240, 40, 240, -5, 120, 36, box{})
	if !ok || x != 40 || y != 900-36 || w != 240 || h != 36 {
		t.Fatalf("no screen = (%v,%v,%v,%v) ok=%v", x, y, w, h, ok)
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
