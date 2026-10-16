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
