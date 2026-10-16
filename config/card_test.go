package config

import (
	"strings"
	"testing"
)

func TestCardEstimateWidthClamps(t *testing.T) {
	c := Card{MinWidth: 100, MaxWidth: 200, CharWidth: 8, PaddingX: 10, FontSize: 12}
	if w := c.EstimateWidth(""); w >= 100 {
		t.Fatalf("empty should not use min width: %d", w)
	}
	long := strings.Repeat("a", 80)
	if w := c.EstimateWidth(long); w != 200 {
		t.Fatalf("long: %d", w)
	}
	hello := c.EstimateWidth("hello")
	if hello >= 100 || hello <= 2*c.PaddingX {
		t.Fatalf("short: %d", hello)
	}
	if floor := c.WidthFloor("hello"); floor != hello {
		t.Fatalf("floor %d want %d", floor, hello)
	}
	if floor := c.WidthFloor(long); floor != 100 {
		t.Fatalf("long floor %d", floor)
	}
}
