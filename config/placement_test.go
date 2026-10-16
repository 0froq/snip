package config

import "testing"

func TestEffectivePlacement(t *testing.T) {
	c := Config{Card: Card{Placement: "near-cursor"}}
	if c.EffectivePlacement() != PlacementNearCursor {
		t.Fatalf("got %q", c.EffectivePlacement())
	}
	c.Card.Placement = "fixed"
	if c.EffectivePlacement() != PlacementFixed {
		t.Fatalf("got %q", c.EffectivePlacement())
	}
	c.Card.Placement = ""
	if c.EffectivePlacement() != PlacementNearSelection {
		t.Fatalf("default: got %q", c.EffectivePlacement())
	}
}
