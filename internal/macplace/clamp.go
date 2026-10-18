package macplace

// Rect is an axis-aligned rectangle in screen coordinates (top-left origin).
type Rect struct {
	X, Y, Width, Height int
}

// Display is one monitor's usable area for clamping card windows.
type Display struct {
	WorkArea Rect
}

const (
	gapBelowSelection = 8
	gapAboveSelection = 8
	cursorOffsetX     = 16
	cursorOffsetY     = 16
)

// PlaceCard chooses a top-left origin near the selection, or near the cursor
// when there is no selection. A failed screen read (no displays, or a zero
// work area) must not move the card to the origin: the anchor is kept and
// only a positive work area is allowed to clamp it.
func PlaceCard(anchor Rect, haveAnchor bool, curX, curY int, haveCursor bool, displays []Display, cardW, cardH int) (x, y int, ok bool) {
	if haveAnchor && anchor.Width > 0 && anchor.Height > 0 {
		wa := WorkAreaForPoint(anchor.X+anchor.Width/2, anchor.Y+anchor.Height/2, displays)
		x, y = PlaceBelowOrAbove(anchor, cardW, cardH, wa)
		return x, y, true
	}
	if haveCursor {
		wa := WorkAreaForPoint(curX, curY, displays)
		x, y = PlaceNearCursor(curX, curY, cardW, cardH, wa)
		return x, y, true
	}
	return 0, 0, false
}

// PlaceBelowOrAbove puts a card just under anchor, or above when there is no room.
func PlaceBelowOrAbove(anchor Rect, cardW, cardH int, screen Rect) (x, y int) {
	x = anchor.X + (anchor.Width-cardW)/2
	y = anchor.Y + anchor.Height + gapBelowSelection
	if screen.Width > 0 && screen.Height > 0 && y+cardH > screen.Y+screen.Height {
		y = anchor.Y - cardH - gapAboveSelection
	}
	return clampOrigin(x, y, cardW, cardH, screen)
}

// PlaceNearCursor offsets the card from the pointer so it does not cover it.
func PlaceNearCursor(px, py int, cardW, cardH int, screen Rect) (x, y int) {
	x = px + cursorOffsetX
	y = py + cursorOffsetY
	return clampOrigin(x, y, cardW, cardH, screen)
}

// FlipAppKitY converts an AppKit bottom-left Y into a top-left Y.
// screenH is the primary display's frame height. A non-positive height
// cannot be flipped: AppKit Y 0 is the bottom edge, and treating a missing
// height as 0 would report that edge as the top.
func FlipAppKitY(appKitY, screenH float64) (int, bool) {
	if screenH <= 0 {
		return 0, false
	}
	return int(screenH - appKitY), true
}

func clampOrigin(x, y, cardW, cardH int, screen Rect) (int, int) {
	// A zero work area is a failed visibleFrame read, not the screen origin.
	if screen.Width <= 0 || screen.Height <= 0 {
		return x, y
	}
	maxX := screen.X + screen.Width - cardW
	maxY := screen.Y + screen.Height - cardH
	if x < screen.X {
		x = screen.X
	}
	if y < screen.Y {
		y = screen.Y
	}
	if x > maxX {
		x = maxX
	}
	if y > maxY {
		y = maxY
	}
	if maxX < screen.X {
		x = screen.X
	}
	if maxY < screen.Y {
		y = screen.Y
	}
	return x, y
}

// WorkAreaForPoint picks the display whose work area contains p, or the nearest.
func WorkAreaForPoint(pX, pY int, displays []Display) Rect {
	usable := make([]Display, 0, len(displays))
	for _, d := range displays {
		if d.WorkArea.Width > 0 && d.WorkArea.Height > 0 {
			usable = append(usable, d)
		}
	}
	if len(usable) == 0 {
		if len(displays) == 0 {
			return Rect{X: 0, Y: 0, Width: 1920, Height: 1080}
		}
		// The screens were enumerated but every visibleFrame was empty.
		return Rect{}
	}
	best := usable[0].WorkArea
	bestDist := distToRect(pX, pY, best)
	for _, d := range usable[1:] {
		wa := d.WorkArea
		if pointInRect(pX, pY, wa) {
			return wa
		}
		if dist := distToRect(pX, pY, wa); dist < bestDist {
			best, bestDist = wa, dist
		}
	}
	return best
}

func pointInRect(x, y int, r Rect) bool {
	return x >= r.X && x < r.X+r.Width && y >= r.Y && y < r.Y+r.Height
}

func distToRect(x, y int, r Rect) int {
	dx := 0
	if x < r.X {
		dx = r.X - x
	} else if x >= r.X+r.Width {
		dx = x - (r.X + r.Width - 1)
	}
	dy := 0
	if y < r.Y {
		dy = r.Y - y
	} else if y >= r.Y+r.Height {
		dy = y - (r.Y + r.Height - 1)
	}
	return dx*dx + dy*dy
}
