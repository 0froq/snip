package config

import (
	"math"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestEstimateSizeOneLine(t *testing.T) {
	c := Default().Card
	layout := c.Layout("hi", 0, 0)
	if layout.VisibleLines != 1 || layout.Truncated {
		t.Fatalf("layout %+v", layout)
	}
	ink, _ := ui.ShapeText("hi", c.cardFont(), 0).Size()
	want := int(math.Ceil(float64(ink))) + 2*c.EffectivePaddingX() + 2*CardBorderPx()
	if layout.Width != want {
		t.Fatalf("width %d want ink+padding %d", layout.Width, want)
	}
	if layout.Width >= c.EffectiveMinWidth() {
		t.Fatalf("short width %d should stay under min %d", layout.Width, c.EffectiveMinWidth())
	}
	chosen := c.Layout("hi", 200, 0)
	if chosen.Width != 200 {
		t.Fatalf("resized width %d", chosen.Width)
	}
}

func TestLayoutWrapsOnlyAtMaxWidth(t *testing.T) {
	c := Default().Card
	short := c.Layout("short note", 0, 0)
	if short.Width >= c.EffectiveMaxWidth() || short.FullLines != 1 {
		t.Fatalf("short layout %+v", short)
	}
	long := c.Layout(strings.Repeat("wide text ", 100), 0, 0)
	if long.Width != c.EffectiveMaxWidth() || long.FullLines <= 1 {
		t.Fatalf("long layout %+v", long)
	}
}

func TestHeightForWidthCJKWraps(t *testing.T) {
	c := Default().Card
	text := strings.Repeat("还有这个样式，我认为既然要做纸色，窗口可以再纸一点", 3)
	layout := c.Layout(text, 220, 0)
	if layout.Truncated || layout.Text != text || strings.Contains(layout.Text, "…") {
		t.Fatalf("layout %+v", layout)
	}
	if layout.Height > c.EffectiveMaxHeight() || layout.FullLines < 2 {
		t.Fatalf("layout %+v", layout)
	}
}

func TestHeightForWidthMixedScript(t *testing.T) {
	c := Default().Card
	c.MaxWidth = 280
	text := "hello 你好 world"
	w := 200
	layout := c.Layout(text, w, 2)
	_, measured := ui.ShapeText(layout.Text, c.cardFont(), c.textWidthForSlip(w)).Size()
	want := int(math.Ceil(float64(measured))) + 2*c.EffectivePaddingY() + 2*CardBorderPx()
	if layout.Height != want {
		t.Fatalf("height %d want measured %d", layout.Height, want)
	}
}

func TestHeightForWidthShrinksWhenWider(t *testing.T) {
	c := Default().Card
	c.MaxWidth = 200
	c.CharWidth = 10
	c.PaddingX = 10
	c.MaxLines = 4
	long := "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"
	narrow := c.Layout(long, 80, 4)
	wide := c.Layout(long, 200, 4)
	if wide.Height >= narrow.Height {
		t.Fatalf("wide height %d should be less than narrow %d", wide.Height, narrow.Height)
	}
}

func TestEstimateSizeCapsLines(t *testing.T) {
	c := Default().Card
	c.MaxWidth = 80
	c.CharWidth = 10
	c.PaddingX = 10
	c.MaxHeight = c.OneLineHeight(80) + 4
	long := "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"
	layout := c.Layout(long, 80, 0)
	if layout.Truncated || layout.Text != long || !layout.Scrolls || layout.Height != c.EffectiveMaxHeight() {
		t.Fatalf("layout %+v max %d", layout, c.EffectiveMaxHeight())
	}
}

func TestLayoutKeepsChosenLines(t *testing.T) {
	c := Default().Card
	text := strings.Repeat("word ", 60)
	one := c.OneLineHeight(200)
	layout := c.LayoutViewport(text, 200, one+12)
	if layout.Truncated || layout.Text != text || layout.Height != one+12 || !layout.Scrolls {
		t.Fatalf("chosen layout %+v", layout)
	}
}

func TestMiddleTruncationShapes(t *testing.T) {
	c := Default().Card
	text := strings.Repeat("还有这个样式，我认为既然要做纸色，窗口可以再纸一点", 4) + "END"
	one := c.OneLineHeight(220)
	for _, n := range []int{1, 2, 3, 4} {
		l := c.LayoutViewport(text, 220, one*n)
		if l.Truncated || l.Text != text || !strings.HasSuffix(l.Text, "END") {
			t.Fatalf("n=%d: %+v", n, l)
		}
		if l.Height > c.EffectiveMaxHeight() || l.Height < one {
			t.Fatalf("n=%d height %d", n, l.Height)
		}
	}
	short := c.Layout("短短", 0, 4)
	if short.Truncated || short.VisibleLines != 1 || short.Scrolls || short.Text != "短短" {
		t.Fatalf("short: %+v", short)
	}
}
