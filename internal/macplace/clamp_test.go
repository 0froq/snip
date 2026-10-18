package macplace

import "testing"

func TestPlaceBelowOrAboveClamps(t *testing.T) {
	screen := Rect{X: 0, Y: 0, Width: 400, Height: 300}
	anchor := Rect{X: 100, Y: 280, Width: 40, Height: 10}
	_, y := PlaceBelowOrAbove(anchor, 120, 44, screen)
	if y+44 > screen.Height {
		t.Fatalf("expected above placement when below overflows, got y=%d", y)
	}
	if y >= anchor.Y {
		t.Fatalf("expected card above anchor, got y=%d anchor.Y=%d", y, anchor.Y)
	}
}

func TestPlaceNearCursorOffsets(t *testing.T) {
	screen := Rect{X: 0, Y: 0, Width: 800, Height: 600}
	x, y := PlaceNearCursor(10, 10, 100, 40, screen)
	if x < 10+cursorOffsetX || y < 10+cursorOffsetY {
		t.Fatalf("unexpected origin %d,%d", x, y)
	}
}

func TestPlaceCardNearSelectionDespiteFailedScreenRead(t *testing.T) {
	anchor := Rect{X: 500, Y: 400, Width: 80, Height: 20}
	screen := Rect{X: 0, Y: 25, Width: 1440, Height: 800}
	x, y, ok := PlaceCard(anchor, true, 0, 0, false, []Display{{WorkArea: screen}}, 200, 48)
	if !ok {
		t.Fatal("selection should place a card")
	}
	// Centered under the selection, not the screen origin.
	if x != 500+(80-200)/2 || y != 400+20+gapBelowSelection {
		t.Fatalf("placed (%d,%d), want under the selection", x, y)
	}
	if x == 0 && y == 0 {
		t.Fatal("card placed at the screen origin")
	}
	// visibleFrame came back empty. Do not clamp to (0,0).
	x, y, ok = PlaceCard(anchor, true, 0, 0, false, []Display{{WorkArea: Rect{}}}, 200, 48)
	if !ok || x == 0 && y == 0 {
		t.Fatalf("zero work area placed (%d,%d) ok=%v", x, y, ok)
	}
	if y != 400+20+gapBelowSelection {
		t.Fatalf("zero work area y=%d, want just below the selection", y)
	}
	// Cursor fallback with a failed screen read stays next to the pointer.
	x, y, ok = PlaceCard(Rect{}, false, 640, 480, true, []Display{{WorkArea: Rect{}}}, 100, 40)
	if !ok || x != 640+cursorOffsetX || y != 480+cursorOffsetY {
		t.Fatalf("cursor placed (%d,%d) ok=%v", x, y, ok)
	}
}

func TestFlippedAppKitOriginIsTheBottomEdge(t *testing.T) {
	// mouseLocation is AppKit bottom-left. (0,0) flipped by a real screen
	// height is the bottom edge, and the cursor fallback then clamps there.
	top, ok := FlipAppKitY(0, 900)
	if !ok || top != 900 {
		t.Fatalf("AppKit origin flips to y=%d ok=%v, want 900", top, ok)
	}
	if _, ok = FlipAppKitY(120, 0); ok {
		t.Fatal("a missing screen height must not flip")
	}
	screen := Rect{X: 0, Y: 25, Width: 1440, Height: 850}
	x, y := PlaceNearCursor(0, top, 100, 40, screen)
	bottom := screen.Y + screen.Height - 40
	if y != bottom || x != cursorOffsetX {
		t.Fatalf("flipped origin placed (%d,%d), want bottom-left (%d,%d)", x, y, cursorOffsetX, bottom)
	}
	// A pointer 200pt from the top of a 900pt screen (AppKit y=700) stays there.
	cursorTop, ok := FlipAppKitY(700, 900)
	if !ok || cursorTop != 200 {
		t.Fatalf("AppKit y=700 flips to %d ok=%v", cursorTop, ok)
	}
	x, y, ok = PlaceCard(Rect{}, false, 400, cursorTop, true, []Display{{WorkArea: screen}}, 100, 40)
	if !ok || x != 400+cursorOffsetX || y != 200+cursorOffsetY {
		t.Fatalf("cursor card = (%d,%d) ok=%v", x, y, ok)
	}
}

func TestWorkAreaForPoint(t *testing.T) {
	displays := []Display{
		{WorkArea: Rect{X: 0, Y: 0, Width: 100, Height: 100}},
		{WorkArea: Rect{X: 200, Y: 0, Width: 100, Height: 100}},
	}
	wa := WorkAreaForPoint(250, 50, displays)
	if wa.X != 200 {
		t.Fatalf("want second display, got %+v", wa)
	}
}
