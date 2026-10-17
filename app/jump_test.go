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
	if b.X+b.W/2 != card.X+card.W || b.Y+b.H/2 != card.Y {
		t.Fatalf("center (%d,%d), card corner (%d,%d)", b.X+b.W/2, b.Y+b.H/2, card.X+card.W, card.Y)
	}
	midX, midY := b.X+b.W/2, b.Y+b.H/2
	bottomLeft := frame{X: b.X, Y: midY, W: b.W / 2, H: b.H / 2}
	if !frameInside(bottomLeft, card) {
		t.Fatalf("bottom-left quarter %+v not on card %+v", bottomLeft, card)
	}
	outside := []frame{
		{X: b.X, Y: b.Y, W: b.W / 2, H: b.H / 2},
		{X: midX, Y: b.Y, W: b.W / 2, H: b.H / 2},
		{X: midX, Y: midY, W: b.W / 2, H: b.H / 2},
	}
	for _, q := range outside {
		if framesOverlap(q, card) {
			t.Fatalf("quadrant %+v overlaps card", q)
		}
		if q.Y+q.H > card.Y && q.X+q.W <= card.X+card.W {
			t.Fatalf("quadrant %+v is not above or right of the card", q)
		}
	}
	overlap := (b.W / 2) * (b.H / 2)
	if overlap*4 != b.W*b.H {
		t.Fatalf("overlap %d is not one quarter of %d", overlap, b.W*b.H)
	}

	odd := BadgeFrame(card, 27)
	if odd.W != 28 || odd.X+odd.W/2 != card.X+card.W || odd.Y+odd.H/2 != card.Y {
		t.Fatalf("odd diameter %+v", odd)
	}
}

func TestPlaceWhichKeyCorners(t *testing.T) {
	screen := frame{X: 10, Y: 30, W: 1000, H: 800}
	const winW, winH, margin = 120, 80, whichKeyMargin
	cases := []struct {
		corner string
		x, y   int
	}{
		{config.WhichKeyTopLeft, screen.X + margin, screen.Y + margin},
		{config.WhichKeyTopRight, screen.X + screen.W - winW - margin, screen.Y + margin},
		{config.WhichKeyBottomLeft, screen.X + margin, screen.Y + screen.H - winH - margin},
		{config.WhichKeyBottomRight, screen.X + screen.W - winW - margin, screen.Y + screen.H - winH - margin},
	}
	for _, tc := range cases {
		x, y := placeWhichKey(screen, tc.corner, winW, winH, margin)
		if x != tc.x || y != tc.y {
			t.Fatalf("%s got (%d,%d) want (%d,%d)", tc.corner, x, y, tc.x, tc.y)
		}
		if x < screen.X+margin || y < screen.Y+margin || x+winW > screen.X+screen.W-margin || y+winH > screen.Y+screen.H-margin {
			t.Fatalf("%s window not inside the margin", tc.corner)
		}
	}
}

func TestWhichKeyItemsLanguage(t *testing.T) {
	cfg := config.Default()
	en := whichKeyItems(cfg)
	cfg.Language = "zh"
	zh := whichKeyItems(cfg)
	if len(en) != 7 || len(zh) != 7 {
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
