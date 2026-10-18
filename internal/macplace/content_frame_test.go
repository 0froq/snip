package macplace

import "testing"

func TestContentFrameMatchingWindow(t *testing.T) {
	// Title-bar inset: the view is shorter and lifted off the bottom of a
	// window whose frame is already the size the user asked for.
	x, y, w, h, changed := contentFrameMatchingWindow(460, 640, 0, 28, 460, 612)
	if !changed || x != 0 || y != 0 || w != 460 || h != 640 {
		t.Fatalf("inset view -> (%v,%v,%v,%v) changed=%v", x, y, w, h, changed)
	}
	// Already the window bounds. Do not write another frame.
	x, y, w, h, changed = contentFrameMatchingWindow(460, 640, 0, 0, 460, 640)
	if changed || x != 0 || y != 0 || w != 460 || h != 640 {
		t.Fatalf("matched view -> (%v,%v,%v,%v) changed=%v", x, y, w, h, changed)
	}
	// A zero window frame is a failed NSRect read, not a content view to apply.
	x, y, w, h, changed = contentFrameMatchingWindow(0, 0, 10, 12, 100, 80)
	if changed || x != 10 || y != 12 || w != 100 || h != 80 {
		t.Fatalf("zero window -> (%v,%v,%v,%v) changed=%v", x, y, w, h, changed)
	}
}
