package app

import "testing"

func TestCardMoveLeavesNativeResizeBorders(t *testing.T) {
	const w, h = 240, 60
	for _, p := range [][2]int{
		{0, 0}, {w - 1, 0}, {0, h - 1}, {w - 1, h - 1},
		{w / 2, 0}, {w / 2, h - 1}, {0, h / 2}, {w - 1, h / 2},
		{nativeResizeMargin - 1, h / 2}, {w - nativeResizeMargin, h / 2},
		{w / 2, nativeResizeMargin - 1}, {w / 2, h - nativeResizeMargin},
		{-1, h / 2}, {w + 1, h / 2},
	} {
		if cardMoveInterior(p[0], p[1], w, h) {
			t.Errorf("edge/corner %v must not start a move", p)
		}
	}
	for _, p := range [][2]int{{w / 2, h / 2}, {nativeResizeMargin, nativeResizeMargin}} {
		if !cardMoveInterior(p[0], p[1], w, h) {
			t.Errorf("interior %v must allow clicks and moves", p)
		}
	}
}

func TestPointerSlop(t *testing.T) {
	slop := pointerSlop * pointerSlop
	if 5*5 >= slop {
		t.Fatal("5pt move should count as tap")
	}
	if 7*7 < slop {
		t.Fatal("7pt move should start drag")
	}
}
