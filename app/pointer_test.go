package app

import "testing"

func TestPointerSlop(t *testing.T) {
	slop := pointerSlop * pointerSlop
	if 5*5 >= slop {
		t.Fatal("5pt move should count as tap")
	}
	if 7*7 < slop {
		t.Fatal("7pt move should start drag")
	}
}
