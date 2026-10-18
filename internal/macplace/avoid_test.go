package macplace

import "testing"

func TestAvoidCardsSameSpot(t *testing.T) {
	screen := Rect{X: 0, Y: 0, Width: 1440, Height: 900}
	const cardW, cardH = 200, 48
	preferX, preferY := 100, 100
	first := Rect{X: preferX, Y: preferY, Width: cardW, Height: cardH}
	// Nothing visible: the near-selection point stays put.
	if x, y := AvoidCards(preferX, preferY, cardW, cardH, nil, screen); x != preferX || y != preferY {
		t.Fatalf("empty field (%d,%d)", x, y)
	}
	x, y := AvoidCards(preferX, preferY, cardW, cardH, []Rect{first}, screen)
	if overlapWithGap(x, y, cardW, cardH, []Rect{first}) != 0 {
		t.Fatalf("second copy still overlaps at (%d,%d)", x, y)
	}
	// Vertical clearance is 48+8, so the first free ring is 64 below.
	if x != preferX || y != preferY+4*avoidStep {
		t.Fatalf("same spot moved to (%d,%d), want (%d,%d)", x, y, preferX, preferY+4*avoidStep)
	}
	// A failed visibleFrame read must not drag that nudge to the origin.
	x, y = AvoidCards(500, 400, cardW, cardH, []Rect{{X: 500, Y: 400, Width: cardW, Height: cardH}}, Rect{})
	if x != 500 || y != 400+4*avoidStep {
		t.Fatalf("zero work area placed (%d,%d)", x, y)
	}
}

func TestAvoidCardsCrowdedCluster(t *testing.T) {
	screen := Rect{X: 0, Y: 0, Width: 1200, Height: 1200}
	const card = 100
	var occupied []Rect
	for _, x := range []int{0, card, 2 * card} {
		for _, y := range []int{0, card, 2 * card} {
			occupied = append(occupied, Rect{X: x, Y: y, Width: card, Height: card})
		}
	}
	preferX, preferY := card, card
	x, y := AvoidCards(preferX, preferY, card, card, occupied, screen)
	if overlapWithGap(x, y, card, card, occupied) != 0 {
		t.Fatalf("crowded spot still overlaps at (%d,%d)", x, y)
	}
	// The cluster covers [0,300). Leaving it downward is the shortest move.
	if x != preferX || y != 308 {
		t.Fatalf("crowded placement (%d,%d), want (%d,308)", x, y, preferX)
	}
	if dist2(x-preferX, y-preferY) > dist2(0, avoidRadius) {
		t.Fatalf("moved farther than the search radius: (%d,%d)", x, y)
	}
}

func TestAvoidCardsScreenEdge(t *testing.T) {
	screen := Rect{X: 10, Y: 20, Width: 400, Height: 300}
	const cardW, cardH = 100, 80
	preferX := screen.X + screen.Width - cardW
	preferY := screen.Y + screen.Height - cardH
	occupied := []Rect{{X: preferX, Y: preferY, Width: cardW, Height: cardH}}
	x, y := AvoidCards(preferX, preferY, cardW, cardH, occupied, screen)
	if x < screen.X || y < screen.Y || x+cardW > screen.X+screen.Width || y+cardH > screen.Y+screen.Height {
		t.Fatalf("left the visible frame: (%d,%d) screen %+v", x, y, screen)
	}
	if overlapWithGap(x, y, cardW, cardH, occupied) != 0 {
		t.Fatalf("edge card still overlaps at (%d,%d)", x, y)
	}
	// Down and right are off the frame. Up clears in 96pt; left needs 112.
	if x != preferX || y != preferY-6*avoidStep {
		t.Fatalf("edge placement (%d,%d), want (%d,%d)", x, y, preferX, preferY-6*avoidStep)
	}
}

func TestAvoidCardsMinimalOverlapWhenNothingIsFree(t *testing.T) {
	screen := Rect{X: 0, Y: 0, Width: 2000, Height: 2000}
	occupied := []Rect{{X: 0, Y: 0, Width: 400, Height: 400}}
	x, y := AvoidCards(0, 0, 400, 400, occupied, screen)
	got := overlapWithGap(x, y, 400, 400, occupied)
	stay := overlapWithGap(0, 0, 400, 400, occupied)
	axis := overlapWithGap(avoidRadius, 0, 400, 400, occupied)
	if got == 0 {
		t.Fatal("a 400pt card cannot clear inside the radius")
	}
	if got >= stay || got > axis {
		t.Fatalf("overlap %d, staying %d, axis shift %d at (%d,%d)", got, stay, axis, x, y)
	}
	if x < 0 || y < 0 || x+400 > screen.Width || y+400 > screen.Height {
		t.Fatalf("overlap fallback left the screen: (%d,%d)", x, y)
	}
}
