package macplace

import (
	"fmt"
	"math"
	"testing"

	"github.com/0froq/snip/config"
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

func TestEightDragDirectionsPlaceTextFromTheTop(t *testing.T) {
	const (
		x, y, w, h                 = 40.0, 80.0, 300.0, 160.0
		dx, dy                     = 48.0, 36.0
		padX, padY, border, margin = 8, 6, 6, 10
		topExtra                   = 24
		docH                       = 500.0
		clipY                      = 40.0
	)
	dirs := []struct {
		name        string
		dir         windowDrag
		originMoves bool
	}{
		{name: "top", dir: windowDragTop, originMoves: false},
		{name: "bottom", dir: windowDragBottom, originMoves: true},
		{name: "left", dir: windowDragLeft, originMoves: true},
		{name: "right", dir: windowDragRight, originMoves: false},
		{name: "top-left", dir: windowDragTopLeft, originMoves: true},
		{name: "top-right", dir: windowDragTopRight, originMoves: false},
		{name: "bottom-left", dir: windowDragBottomLeft, originMoves: true},
		{name: "bottom-right", dir: windowDragBottomRight, originMoves: true},
	}
	left, top, right, bottom := config.TextFrameInsets(padX, padY, border, margin, topExtra)
	var topLay, bottomLay, leftLay, rightLay draggedCardLayout
	for _, tc := range dirs {
		lay := layoutDraggedCard(x, y, w, h, dx, dy, tc.dir, padX, padY, border, margin, topExtra, docH, clipY)
		if !lay.OK {
			t.Fatalf("%s: text frame was rejected", tc.name)
		}
		moved := lay.WinX != x || lay.WinY != y
		if moved != tc.originMoves {
			t.Fatalf("%s origin moved=%v, want %v (%v,%v) -> (%v,%v)", tc.name, moved, tc.originMoves, x, y, lay.WinX, lay.WinY)
		}
		if lay.ContentX != 0 || lay.ContentY != 0 || lay.ContentW != lay.WinW || lay.ContentH != lay.WinH {
			t.Fatalf("%s content view (%v,%v,%v,%v), want (0,0,%v,%v)", tc.name, lay.ContentX, lay.ContentY, lay.ContentW, lay.ContentH, lay.WinW, lay.WinH)
		}
		if lay.PadLeft != left || lay.PadTop != top || lay.PadRight != right || lay.PadBottom != bottom {
			t.Fatalf("%s padding (%d,%d,%d,%d), want (%d,%d,%d,%d)", tc.name, lay.PadLeft, lay.PadTop, lay.PadRight, lay.PadBottom, left, top, right, bottom)
		}
		if lay.TextX != float64(left) || lay.TextY != float64(bottom) {
			t.Fatalf("%s text origin (%v,%v), want padding (%d,%d)", tc.name, lay.TextX, lay.TextY, left, bottom)
		}
		if lay.ContentH-(lay.TextY+lay.TextH) != float64(top) {
			t.Fatalf("%s top gap %v, want %d", tc.name, lay.ContentH-(lay.TextY+lay.TextH), top)
		}
		if lay.TextW != lay.WinW-float64(left+right) || lay.TextH != lay.WinH-float64(top+bottom) {
			t.Fatalf("%s text size (%v,%v)", tc.name, lay.TextW, lay.TextH)
		}
		if lay.ContainerW != lay.TextW-2*config.TextContainerInsetX {
			t.Fatalf("%s container %v, text width %v", tc.name, lay.ContainerW, lay.TextW)
		}
		if lay.ScrollX != 0 {
			t.Fatalf("%s horizontal scroll %v", tc.name, lay.ScrollX)
		}
		wantY := clampClipOrigin(clipY, docH, lay.TextH)
		if lay.ScrollY != wantY {
			t.Fatalf("%s scroll %v, want %v", tc.name, lay.ScrollY, wantY)
		}
		if tc.originMoves {
			// The content view can slide by the window-origin delta before
			// its size updates. That is the same relayout as a size change.
			ox, oy := 0.0, 0.0
			if lay.WinX != x {
				ox = x - lay.WinX
			}
			if lay.WinY != y {
				oy = y - lay.WinY
			}
			if contentSizeChanged(w, h, w, h) || !contentFrameChanged(0, 0, w, h, ox, oy, w, h) {
				t.Fatalf("%s origin slide (%v,%v) was not a relayout", tc.name, ox, oy)
			}
			if lay.WinY != y {
				shifted := clampClipOrigin(clipY-(lay.WinY-y), docH, lay.TextH)
				if shifted == wantY || lay.ScrollY == shifted {
					t.Fatalf("%s scroll followed the window origin (%v, kept %v)", tc.name, shifted, lay.ScrollY)
				}
			}
		}
		fits := layoutDraggedCard(x, y, w, h, dx, dy, tc.dir, padX, padY, border, margin, topExtra, 10, 80)
		if !fits.OK || fits.ScrollY != 0 {
			t.Fatalf("%s text that fits scrolled to %v", tc.name, fits.ScrollY)
		}
		switch tc.dir {
		case windowDragTop:
			topLay = lay
		case windowDragBottom:
			bottomLay = lay
		case windowDragLeft:
			leftLay = lay
		case windowDragRight:
			rightLay = lay
		case windowDragTopLeft, windowDragTopRight, windowDragBottomLeft, windowDragBottomRight:
		default:
			panic(fmt.Sprintf("snip: unhandled window drag %d", int(tc.dir)))
		}
	}
	assertSameText(t, "top/bottom", topLay, bottomLay)
	assertSameText(t, "left/right", leftLay, rightLay)
	if contentFrameChanged(0, 0, w, h, 0, 0, w, h) {
		t.Fatal("a content view that already fills the window relayouts again")
	}
	if contentFrameChanged(0, 0, w, h, 0, 0, 0, h) || contentFrameChanged(0, 0, w, h, 0, 0, w, -1) {
		t.Fatal("a failed size read relayouts")
	}
	// Same resulting size, whether or not the origin moved.
	grown := layoutCardFrame(40, 40, 348, 196, padX, padY, border, margin, topExtra, docH, clipY, true, clipY+36, -48, -40)
	if !grown.OK || grown.ContentX != 0 || grown.ContentY != 0 || grown.ScrollY != clampClipOrigin(clipY, docH, grown.TextH) {
		t.Fatalf("pinned grown card %+v", grown)
	}
}

func assertSameText(t *testing.T, name string, a, b draggedCardLayout) {
	t.Helper()
	if a.TextX != b.TextX || a.TextY != b.TextY || a.TextW != b.TextW || a.TextH != b.TextH ||
		a.ContainerW != b.ContainerW || a.ScrollX != b.ScrollX || a.ScrollY != b.ScrollY ||
		a.PadTop != b.PadTop || a.ContentW != b.ContentW || a.ContentH != b.ContentH {
		t.Fatalf("%s placed text differently: %+v vs %+v", name, a, b)
	}
	if a.WinX == b.WinX && a.WinY == b.WinY {
		t.Fatalf("%s windows did not differ in origin", name)
	}
}

func TestLiveOriginChangeRelayoutsWhenContentFrameStaysPut(t *testing.T) {
	const x, y, w, h = 40.0, 80.0, 300.0, 160.0
	// Borderless content view: origin stays (0, 0). The window origin is
	// what a left or bottom drag changes.
	nx, ny, nw, nh := dragWindow(x, y, w, h, 48, 36, windowDragBottomLeft)
	if contentFrameChanged(0, 0, w, h, 0, 0, w, h) {
		t.Fatal("an unchanged content view frame looked like a left/bottom drag")
	}
	if !liveStepNeedsRelayout(true, x, y, w, h, nx, ny, nw, nh) {
		t.Fatal("a live left/bottom drag did not relayout")
	}
	// Top/right change the size and leave the origin. Same decision.
	tx, ty, tw, th := dragWindow(x, y, w, h, 48, 36, windowDragTopRight)
	if tx != x || ty != y || !liveStepNeedsRelayout(true, x, y, w, h, tx, ty, tw, th) {
		t.Fatalf("top/right (%v,%v,%v,%v)", tx, ty, tw, th)
	}
	// Dragging the card changes the origin and keeps the size.
	if liveStepNeedsRelayout(false, x, y, w, h, x+20, y-10, w, h) {
		t.Fatal("a pure move relayouts")
	}
	if liveStepNeedsRelayout(true, x, y, w, h, x, y, 0, h) {
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
