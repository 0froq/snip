package macplace

const (
	// avoidGap is the clear space left between a new card and any visible one.
	avoidGap = 8
	// avoidStep is the spacing of the search spiral. Small enough that the
	// chosen spot stays near the selection, large enough to leave the gap.
	avoidStep = 16
	// avoidRadius caps how far a card may move to get clear. Past this, the
	// search keeps the candidate with the smallest overlap.
	avoidRadius = 320
)

// AvoidCards moves a new card off existing visible cards.
//
// preferX, preferY is the near-selection (or cursor, or cascade) origin.
// Candidates are a square spiral of avoidStep offsets out to avoidRadius,
// starting downward. Each candidate is clamped into screen when that rect
// has a positive size, so the card stays inside the visible frame. A zero
// screen is a failed visibleFrame read and does not pull the card to the
// origin.
//
// The free spot (no intersection with any card expanded by avoidGap) with
// the smallest displacement wins. When every candidate still overlaps, the
// smallest overlap area wins, and equal areas keep the smaller displacement.
func AvoidCards(preferX, preferY, cardW, cardH int, occupied []Rect, screen Rect) (x, y int) {
	if cardW < 1 {
		cardW = 1
	}
	if cardH < 1 {
		cardH = 1
	}
	bestX, bestY := clampOrigin(preferX, preferY, cardW, cardH, screen)
	bestOverlap := overlapWithGap(bestX, bestY, cardW, cardH, occupied)
	bestDist := dist2(bestX-preferX, bestY-preferY)
	for r := avoidStep; r <= avoidRadius; r += avoidStep {
		for _, off := range ringOffsets(r) {
			cx, cy := clampOrigin(preferX+off[0], preferY+off[1], cardW, cardH, screen)
			overlap := overlapWithGap(cx, cy, cardW, cardH, occupied)
			dist := dist2(cx-preferX, cy-preferY)
			if overlap < bestOverlap || (overlap == bestOverlap && dist < bestDist) {
				bestX, bestY = cx, cy
				bestOverlap = overlap
				bestDist = dist
			}
		}
	}
	return bestX, bestY
}

// ringOffsets walks the square ring at chebyshev radius r, starting on the
// bottom edge so a downward nudge wins a tie with an upward one.
func ringOffsets(r int) [][2]int {
	n := 8 * r / avoidStep
	pts := make([][2]int, 0, n)
	for x := -r; x <= r; x += avoidStep {
		pts = append(pts, [2]int{x, r})
	}
	for y := r - avoidStep; y >= -r; y -= avoidStep {
		pts = append(pts, [2]int{r, y})
	}
	for x := r - avoidStep; x >= -r; x -= avoidStep {
		pts = append(pts, [2]int{x, -r})
	}
	for y := -r + avoidStep; y <= r-avoidStep; y += avoidStep {
		pts = append(pts, [2]int{-r, y})
	}
	return pts
}

func overlapWithGap(x, y, cardW, cardH int, occupied []Rect) int {
	card := Rect{X: x, Y: y, Width: cardW, Height: cardH}
	total := 0
	for _, o := range occupied {
		inflated := Rect{
			X:      o.X - avoidGap,
			Y:      o.Y - avoidGap,
			Width:  o.Width + 2*avoidGap,
			Height: o.Height + 2*avoidGap,
		}
		total += intersectionArea(card, inflated)
	}
	return total
}

func intersectionArea(a, b Rect) int {
	x1 := a.X
	if b.X > x1 {
		x1 = b.X
	}
	y1 := a.Y
	if b.Y > y1 {
		y1 = b.Y
	}
	x2 := a.X + a.Width
	if right := b.X + b.Width; right < x2 {
		x2 = right
	}
	y2 := a.Y + a.Height
	if bottom := b.Y + b.Height; bottom < y2 {
		y2 = bottom
	}
	if x2 <= x1 || y2 <= y1 {
		return 0
	}
	return (x2 - x1) * (y2 - y1)
}

func dist2(dx, dy int) int {
	return dx*dx + dy*dy
}
