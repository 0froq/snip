package app

import "testing"

func TestResizeWidthRightAndLeft(t *testing.T) {
	w, shift := resizeWidth(200, 40, 1, 120, 480)
	if w != 240 || shift != 0 {
		t.Fatalf("east %d shift %d", w, shift)
	}
	w, shift = resizeWidth(200, 40, -1, 120, 480)
	if w != 160 || shift != 40 {
		t.Fatalf("west %d shift %d", w, shift)
	}
	w, shift = resizeWidth(200, 500, 1, 120, 480)
	if w != 480 || shift != 0 {
		t.Fatalf("cap %d", w)
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
