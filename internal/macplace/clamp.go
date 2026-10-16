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

// PlaceBelowOrAbove puts a card just under anchor, or above when there is no room.
func PlaceBelowOrAbove(anchor Rect, cardW, cardH int, screen Rect) (x, y int) {
	x = anchor.X + (anchor.Width-cardW)/2
	y = anchor.Y + anchor.Height + gapBelowSelection
	if y+cardH > screen.Y+screen.Height {
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

func clampOrigin(x, y, cardW, cardH int, screen Rect) (int, int) {
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
	if len(displays) == 0 {
		return Rect{X: 0, Y: 0, Width: 1920, Height: 1080}
	}
	best := displays[0].WorkArea
	bestDist := distToRect(pX, pY, best)
	for _, d := range displays[1:] {
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
