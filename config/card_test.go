package config

import (
	"strings"
	"testing"
)

func TestCardEstimateWidthClamps(t *testing.T) {
	c := Card{MinWidth: 100, MaxWidth: 200, CharWidth: 8, PaddingX: 10}
	if w := c.EstimateWidth(""); w != 100 {
		t.Fatalf("empty: %d", w)
	}
	long := strings.Repeat("a", 80)
	if w := c.EstimateWidth(long); w != 200 {
		t.Fatalf("long: %d", w)
	}
	if w := c.EstimateWidth("hello"); w < 100 || w > 200 {
		t.Fatalf("short: %d", w)
	}
}
