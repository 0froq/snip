package config

import "testing"

func TestEstimateSizeOneLine(t *testing.T) {
	c := Default().Card
	_, h := c.EstimateSize("hi")
	one := c.EffectiveLineHeight() + 2*c.EffectivePaddingY()
	if h != one {
		t.Fatalf("one line height %d want %d", h, one)
	}
}

func TestHeightForWidthShrinksWhenWider(t *testing.T) {
	c := Default().Card
	c.MaxWidth = 200
	c.CharWidth = 10
	c.PaddingX = 10
	c.MaxLines = 4
	long := "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"
	narrow := c.HeightForWidth(long, 80)
	wide := c.HeightForWidth(long, 200)
	if wide >= narrow {
		t.Fatalf("wide height %d should be less than narrow %d", wide, narrow)
	}
}

func TestEstimateSizeCapsLines(t *testing.T) {
	c := Default().Card
	c.MaxWidth = 80
	c.CharWidth = 10
	c.PaddingX = 10
	c.MaxLines = 2
	long := "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"
	_, h := c.EstimateSize(long)
	maxH := 2*c.EffectiveLineHeight() + 2*c.EffectivePaddingY()
	if h > maxH {
		t.Fatalf("height %d over %d", h, maxH)
	}
}

func TestEffectiveToggleLegacy(t *testing.T) {
	s := Shortcuts{ShowAll: "Cmd+Shift+S"}
	if s.EffectiveToggle() != "Cmd+Shift+S" {
		t.Fatal(s.EffectiveToggle())
	}
	s.Toggle = "Cmd+Shift+A"
	if s.EffectiveToggle() != "Cmd+Shift+A" {
		t.Fatal(s.EffectiveToggle())
	}
}
