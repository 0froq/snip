package app

import (
	"reflect"
	"testing"

	"github.com/0froq/snip/config"
	"github.com/egoist/mygo/ui"
)

func TestAssignJumpLabelsDigitsFirst(t *testing.T) {
	ids := []string{"newest", "older", "oldest"}
	got := assignJumpLabels(ids, nil, jumpKeyPool)
	want := []string{"1", "2", "3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("labels %v", got)
	}

	many := make([]string, 11)
	for i := range many {
		many[i] = string(rune('a' + i))
	}
	got = assignJumpLabels(many, nil, jumpKeyPool)
	if got[0] != "1" || got[8] != "9" || got[9] != "0" || got[10] != "a" {
		t.Fatalf("digit order %v", got[:11])
	}
}

func TestAssignJumpLabelsSkipsBoundFollowUps(t *testing.T) {
	ids := []string{"c1", "c2", "c3", "c4"}
	bound := map[string]struct{}{"1": {}, "J": {}, "2": {}, "a": {}}
	got := assignJumpLabels(ids, bound, jumpKeyPool)
	want := []string{"3", "4", "5", "6"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("labels %v", got)
	}

	allDigits := map[string]struct{}{}
	for _, d := range []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "0"} {
		allDigits[d] = struct{}{}
	}
	if got := assignJumpLabels([]string{"x"}, allDigits, jumpKeyPool); got[0] != "a" {
		t.Fatalf("after digits, got %v", got)
	}

	cfg := config.Default()
	got = assignJumpLabels(ids, boundLeaderKeys(cfg.Bindings), jumpKeyPool)
	if !reflect.DeepEqual(got, []string{"1", "2", "3", "4"}) {
		t.Fatalf("default bindings still leave digits free: %v", got)
	}
	for _, item := range whichKeyItems(cfg) {
		for _, label := range got {
			if item.Key == label {
				t.Fatalf("label %q collides with leader action %q", label, item.Name)
			}
		}
	}
}

func TestModifiedFollowUpDoesNotStealJumpKey(t *testing.T) {
	onlyShift := boundLeaderKeys(config.BindingList{
		{Action: "clear_cards", Keys: "leader shift+d"},
	})
	if _, ok := onlyShift["d"]; ok {
		t.Fatalf("shift+d occupied d: %v", onlyShift)
	}
	if got := assignJumpLabels([]string{"card"}, onlyShift, jumpKeyPool); got[0] != "1" {
		t.Fatalf("modified follow-up changed the pool: %v", got)
	}
	both := boundLeaderKeys(config.BindingList{
		{Action: "delete", Keys: "leader d"},
		{Action: "clear_cards", Keys: "leader Shift+D"},
	})
	if _, ok := both["d"]; !ok {
		t.Fatal("plain d should occupy the jump letter")
	}
	if _, ok := both["shift+d"]; ok {
		t.Fatal("modified chord stored as a jump letter")
	}
	cfg := config.Default()
	var sawD, sawShift bool
	for _, item := range whichKeyItems(cfg) {
		if item.Key == "d" {
			sawD = true
		}
		if item.Key == "Shift+D" {
			sawShift = true
		}
	}
	if !sawD || !sawShift {
		t.Fatal("which-key collapsed shift+d and d")
	}
}

func TestAssignJumpLabelsStableOrder(t *testing.T) {
	ids := []string{"c", "b", "a"}
	bound := map[string]struct{}{"s": {}, "v": {}, "S": {}}
	first := assignJumpLabels(ids, bound, jumpKeyPool)
	second := assignJumpLabels(ids, bound, jumpKeyPool)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("unstable %v vs %v", first, second)
	}
	if !reflect.DeepEqual(first, []string{"1", "2", "3"}) {
		t.Fatalf("order follows ids, got %v", first)
	}
	// Map iteration order must not change which keys are skipped.
	again := map[string]struct{}{"v": {}, "s": {}}
	if !reflect.DeepEqual(first, assignJumpLabels(ids, again, jumpKeyPool)) {
		t.Fatal("bound map order changed labels")
	}
}

func TestAssignJumpLabelsTwoKey(t *testing.T) {
	pool := []string{"1", "2", "a"}
	ids := []string{"a", "b", "c"}
	single := assignJumpLabels(ids, nil, pool)
	if !reflect.DeepEqual(single, []string{"1", "2", "a"}) {
		t.Fatalf("single %v", single)
	}
	overflow := append(append([]string{}, ids...), "d")
	got := assignJumpLabels(overflow, nil, pool)
	want := []string{"11", "12", "1a", "21"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("two-key %v", got)
	}
	// A bound pair is skipped; singles are not mixed in once two-key starts.
	bound := map[string]struct{}{"12": {}, "z": {}}
	got = assignJumpLabels(overflow, bound, pool)
	want = []string{"11", "1a", "21", "22"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("skipped pair %v", got)
	}
}

func TestBadgeFrameCentersOnTopRight(t *testing.T) {
	card := frame{X: 100, Y: 200, W: 80, H: 40}
	b := BadgeFrame(card, BadgeDiameter)
	if b.W != BadgeDiameter || b.H != BadgeDiameter {
		t.Fatalf("size %d x %d", b.W, b.H)
	}
	if b.W%pixelDot != 0 || (b.W/pixelDot)%2 == 0 {
		t.Fatalf("diameter %d is not an odd number of dots", b.W)
	}
	if (b.X-card.X)%pixelDot != 0 || (b.Y-card.Y)%pixelDot != 0 {
		t.Fatalf("origin %+v is off the card grid %+v", b, card)
	}
	cx, cy := badgeCenterDot(b)
	cornerX := card.X + card.W - pixelDot
	if cx != cornerX || cy != card.Y {
		t.Fatalf("center dot (%d,%d), card corner dot (%d,%d)", cx, cy, cornerX, card.Y)
	}
	// The center dot is the shared corner. Everything below-left of it,
	// including that dot, lies on the card. The other three quadrants do not.
	onCard := frame{X: b.X, Y: cy, W: cx + pixelDot - b.X, H: b.Y + b.H - cy}
	if !frameInside(onCard, card) {
		t.Fatalf("bottom-left %+v not on card %+v", onCard, card)
	}
	outside := []frame{
		{X: b.X, Y: b.Y, W: cx + pixelDot - b.X, H: cy - b.Y},
		{X: cx + pixelDot, Y: b.Y, W: b.X + b.W - (cx + pixelDot), H: cy + pixelDot - b.Y},
		{X: cx + pixelDot, Y: cy + pixelDot, W: b.X + b.W - (cx + pixelDot), H: b.Y + b.H - (cy + pixelDot)},
	}
	for _, q := range outside {
		if framesOverlap(q, card) {
			t.Fatalf("quadrant %+v overlaps card", q)
		}
	}

	odd := BadgeFrame(card, 27)
	ox, oy := badgeCenterDot(odd)
	if odd.W != 15*pixelDot || ox != cornerX || oy != card.Y {
		t.Fatalf("odd diameter %+v", odd)
	}

	skewed := frame{X: 101, Y: 201, W: 81, H: 41}
	snapped := BadgeFrame(skewed, BadgeDiameter)
	if (snapped.X-skewed.X)%pixelDot != 0 || (snapped.Y-skewed.Y)%pixelDot != 0 {
		t.Fatalf("skewed origin %+v off grid of %+v", snapped, skewed)
	}
	sx, sy := badgeCenterDot(snapped)
	wantX := skewed.X + (skewed.W/pixelDot)*pixelDot - pixelDot
	if sx != wantX || sy != skewed.Y {
		t.Fatalf("skewed center (%d,%d), want (%d,%d)", sx, sy, wantX, skewed.Y)
	}
}

func badgeCenterDot(b frame) (x, y int) {
	return b.X + (b.W/pixelDot/2)*pixelDot, b.Y + (b.H/pixelDot/2)*pixelDot
}

func TestPlaceWhichKeyAnchor(t *testing.T) {
	screen := frame{X: 10, Y: 30, W: 1000, H: 800}
	const winW, winH, margin = 120, 80, whichKeyMargin
	cases := []struct {
		name       string
		xPct, yPct int
		ax, ay     string
		x, y       int
	}{
		{"top-left", 0, 0, config.AlignStart, config.AlignStart, screen.X + margin, screen.Y + margin},
		{"top-right", 100, 0, config.AlignEnd, config.AlignStart, screen.X + screen.W - winW - margin, screen.Y + margin},
		{"bottom-left", 0, 100, config.AlignStart, config.AlignEnd, screen.X + margin, screen.Y + screen.H - winH - margin},
		{"bottom-right", 100, 100, config.AlignEnd, config.AlignEnd, screen.X + screen.W - winW - margin, screen.Y + screen.H - winH - margin},
	}
	for _, tc := range cases {
		x, y := placeWhichKey(screen, tc.xPct, tc.yPct, tc.ax, tc.ay, winW, winH, margin)
		if x != tc.x || y != tc.y {
			t.Fatalf("%s got (%d,%d) want (%d,%d)", tc.name, x, y, tc.x, tc.y)
		}
	}
	innerW := screen.W - 2*margin
	innerH := screen.H - 2*margin
	cx := screen.X + margin + innerW/2
	cy := screen.Y + margin + innerH/2
	x, y := placeWhichKey(screen, 50, 50, config.AlignCenter, config.AlignCenter, winW, winH, margin)
	if x != cx-winW/2 || y != cy-winH/2 {
		t.Fatalf("center got (%d,%d) want (%d,%d)", x, y, cx-winW/2, cy-winH/2)
	}
	x, y = placeWhichKey(screen, 100, 100, config.AlignEnd, config.AlignEnd, screen.W+40, screen.H+40, margin)
	if x != screen.X || y != screen.Y {
		t.Fatalf("oversized panel (%d,%d)", x, y)
	}
	x, y = placeWhichKey(screen, 0, 0, config.AlignEnd, config.AlignEnd, winW, winH, margin)
	if x < screen.X || y < screen.Y || x+winW > screen.X+screen.W || y+winH > screen.Y+screen.H {
		t.Fatalf("clamped panel (%d,%d) leaves the screen", x, y)
	}
}

func TestWhichKeyItemsLanguage(t *testing.T) {
	cfg := config.Default()
	en := whichKeyItems(cfg)
	cfg.Language = "zh"
	zh := whichKeyItems(cfg)
	if len(en) != 8 || len(zh) != 8 {
		t.Fatalf("en %d zh %d", len(en), len(zh))
	}
	var enPaste, zhPaste string
	for i := range en {
		if en[i].Key != zh[i].Key {
			t.Fatalf("key order %q vs %q", en[i].Key, zh[i].Key)
		}
		if en[i].Key == "v" {
			enPaste, zhPaste = en[i].Name, zh[i].Name
		}
	}
	if enPaste != "paste" || zhPaste != "粘贴" {
		t.Fatalf("paste hint en %q zh %q", enPaste, zhPaste)
	}
}

func TestBadgeGlyphNudgeLiftsFigures(t *testing.T) {
	// Extra ascent above a 0.70em cap leaves the figure low in a centered line box.
	n := badgeGlyphNudge(ui.FontMetrics{Ascent: 16, Descent: 4}, 14)
	if n >= 0 || n < -3 {
		t.Fatalf("nudge %v should be a small upward shift", n)
	}
	// Line gap does not move the baseline relative to the line-box center.
	if badgeGlyphNudge(ui.FontMetrics{Ascent: 16, Descent: 4, LineGap: 4}, 14) != n {
		t.Fatal("line gap changed the nudge")
	}
	if badgeGlyphNudge(ui.FontMetrics{}, 14) != 0 {
		t.Fatal("missing metrics should not nudge")
	}
}

func frameInside(inner, outer frame) bool {
	return inner.X >= outer.X && inner.Y >= outer.Y &&
		inner.X+inner.W <= outer.X+outer.W && inner.Y+inner.H <= outer.Y+outer.H
}

func framesOverlap(a, b frame) bool {
	return a.X < b.X+b.W && a.X+a.W > b.X && a.Y < b.Y+b.H && a.Y+a.H > b.Y
}
