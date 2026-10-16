package config

import (
	"math"
	"testing"
)

func TestEstimateSizeOneLine(t *testing.T) {
	c := Default().Card
	lines, h := c.shapedHeight("hi", c.EstimateWidth("hi"))
	if lines != 1 {
		t.Fatalf("lines %d", lines)
	}
	two := 2*int(math.Ceil(float64(c.LineBox()))) + 2*c.EffectivePaddingY() + 2*CardBorderPx()
	if h >= two {
		t.Fatalf("one line height %d reaches a second line at %d", h, two)
	}
}

func TestHeightForWidthCJKWraps(t *testing.T) {
	c := Default().Card
	text := "还有这个样式，我认为既然要做纸色，窗口可以再纸一点"
	w := 220
	lines, _ := c.shapedHeight(text, w)
	if lines < 2 {
		t.Fatalf("expected at least 2 wrapped lines for CJK, got %d at width %d", lines, w)
	}
	h := c.HeightForWidth(text, w)
	two := 2*int(math.Ceil(float64(c.LineBox()))) + 2*c.EffectivePaddingY() + 2*CardBorderPx()
	three := 3*int(math.Ceil(float64(c.LineBox()))) + 2*c.EffectivePaddingY() + 2*CardBorderPx()
	if h < two {
		t.Fatalf("height %d too small for two lines (need >= %d)", h, two)
	}
	if h >= three {
		t.Fatalf("height %d includes a blank third line", h)
	}
}

func TestHeightForWidthMixedScript(t *testing.T) {
	c := Default().Card
	c.MaxWidth = 280
	text := "hello 你好 world"
	w := 200
	lines, _ := c.shapedHeight(text, w)
	if lines < 1 {
		t.Fatal("expected at least one line")
	}
	h := c.HeightForWidth(text, w)
	min := lines*c.EffectiveLineHeight() + 2*c.EffectivePaddingY() + 2*CardBorderPx()
	if h < min {
		t.Fatalf("height %d below shaped minimum %d", h, min)
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
	maxH := 2*int(math.Ceil(float64(c.LineBox()))) + 2*c.EffectivePaddingY() + 2*CardBorderPx()
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
