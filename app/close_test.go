package app

import "testing"

func TestCloseButtonWinsOverDragAndResize(t *testing.T) {
	const w, h = 240, 60
	cx := closeButtonX + closeButtonSize/2
	cy := closeButtonY + closeButtonSize/2
	kind, _ := cardHitTest(cx, cy, w, h)
	if kind != cardHitClose {
		t.Fatalf("center hit %v", kind)
	}
	// The circle overlaps the left resize band. That overlap closes.
	kind, _ = cardHitTest(closeButtonX, cy, w, h)
	if kind != cardHitClose {
		t.Fatalf("left edge of the circle hit %v", kind)
	}
	// The rest of the left band still resizes, and the interior still moves.
	kind, edge := cardHitTest(closeButtonX-1, cy, w, h)
	if kind != cardHitResize || edge&edgeLeft == 0 {
		t.Fatalf("beside the circle: kind %v edge %d", kind, edge)
	}
	kind, edge = cardHitTest(0, h/2, w, h)
	if kind != cardHitResize || edge&edgeLeft == 0 {
		t.Fatalf("left edge: kind %v edge %d", kind, edge)
	}
	kind, edge = cardHitTest(w/2, h-1, w, h)
	if kind != cardHitResize || edge&edgeBottom == 0 {
		t.Fatalf("bottom edge: kind %v edge %d", kind, edge)
	}
	kind, _ = cardHitTest(w/2, h/2, w, h)
	if kind != cardHitMove {
		t.Fatalf("interior hit %v", kind)
	}
	if closeButtonContains(cx, cy+closeButtonSize) {
		t.Fatal("point below the circle is inside it")
	}
}
