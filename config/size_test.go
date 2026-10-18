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
	chrome := 2*float64(c.FrameInsetX()) + 2*TextContainerInsetX + 2*LineFragmentPadding
	want := int(math.Ceil(float64(ink) + chrome + lineFitSlack))
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

func TestVoidJingleDeployFitsOnOneLine(t *testing.T) {
	c := Default().Card
	const text = "void-jingle deploy"
	layout := c.Layout(text, 0, 0)
	if layout.FullLines != 1 || layout.Scrolls || layout.Width >= c.EffectiveMaxWidth() {
		t.Fatalf("layout %+v max %d", layout, c.EffectiveMaxWidth())
	}
	ink, _ := ui.ShapeText(text, c.cardFont(), 0).Size()
	chrome := 2*float64(c.FrameInsetX()) + 2*TextContainerInsetX + 2*LineFragmentPadding
	want := int(math.Ceil(float64(ink) + chrome + lineFitSlack))
	if layout.Width != want {
		t.Fatalf("width %d want %d ink %v chrome %v", layout.Width, want, ink, chrome)
	}
	// The pixel frame is part of the horizontal inset. A fragment that
	// drops it is a word short of this line.
	inset := c.EffectivePaddingX() + cardBorderPx
	if CardResizeMargin > inset {
		inset = CardResizeMargin
	}
	if c.FrameInsetX() != inset {
		t.Fatalf("frame inset %d want %d", c.FrameInsetX(), inset)
	}
	if fragment := float64(layout.Width) - chrome; fragment < float64(ink)+lineFitSlack-1e-6 {
		t.Fatalf("fragment %v ink %v slack %v", fragment, ink, lineFitSlack)
	}
}

func TestSingleAndMultiLineViewports(t *testing.T) {
	c := Default().Card
	one := c.Layout("void-jingle deploy", 0, 0)
	if one.FullLines != 1 || one.Scrolls || one.Width >= c.EffectiveMaxWidth() {
		t.Fatalf("single: %+v", one)
	}
	left, top, right, bottom := c.edgeInsets()
	if top < CardTitleStripPx || bottom < cardBorderPx || left < cardBorderPx || right != left {
		t.Fatalf("chrome LTRB %d %d %d %d", left, top, right, bottom)
	}
	viewH := one.Height - top - bottom
	if viewH < c.LineBoxPx()+int(math.Ceil(2*TextContainerInsetY+lineFitSlack)) {
		t.Fatalf("single viewport %d line %d", viewH, c.LineBoxPx())
	}
	frag := float64(one.Width-left-right) - 2*TextContainerInsetX - 2*LineFragmentPadding
	_, shapedH := ui.ShapeText(one.Text, c.cardFont(), float32(frag)).Size()
	if float64(viewH) < float64(math.Ceil(float64(shapedH)))+2*TextContainerInsetY+lineFitSlack-1e-6 {
		t.Fatalf("viewport %d shaped %v", viewH, shapedH)
	}

	multi := c.Layout("void-jingle deploy\nagain\nand more", 0, 0)
	if multi.FullLines != 3 || multi.Scrolls || multi.Height <= one.Height {
		t.Fatalf("multi: %+v one %d", multi, one.Height)
	}
	if multi.Height-one.Height != 2*c.LineBoxPx() {
		t.Fatalf("multi grew %d, want %d", multi.Height-one.Height, 2*c.LineBoxPx())
	}
	if multi.Height-top-bottom < 3*c.LineBoxPx() {
		t.Fatalf("multi viewport %d", multi.Height-top-bottom)
	}

	c.MaxHeight = one.Height
	over := c.Layout("void-jingle deploy\nagain\nand more\nand past the max", 0, 0)
	if over.FullLines <= 3 || !over.Scrolls || over.Height != c.EffectiveMaxHeight() {
		t.Fatalf("over max: %+v max %d", over, c.EffectiveMaxHeight())
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
	want := c.windowHeight(c.textBlockPx(layout.FullLines, measured))
	if want > c.EffectiveMaxHeight() {
		want = c.EffectiveMaxHeight()
	}
	if layout.Height != want {
		t.Fatalf("height %d want measured %d", layout.Height, want)
	}
	if layout.Height-2*c.FrameInsetY() < layout.FullLines*c.LineBoxPx() {
		t.Fatalf("viewport clips the line box: %+v", layout)
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

func TestCardHeightFitsOneLineThreeLinesAndMax(t *testing.T) {
	c := Default().Card
	c.FontSize = 14
	one := c.Layout("#00000", 240, 0)
	line := c.LineBoxPx()
	if one.FullLines != 1 || one.Scrolls || one.Truncated || one.Text != "#00000" {
		t.Fatalf("one line: %+v", one)
	}
	if one.Height != c.windowHeight(c.textBlockPx(1, c.lineBox())) {
		t.Fatalf("one line height %d", one.Height)
	}
	if one.Height-c.TopInset()-c.FrameInsetY() < line+int(math.Ceil(lineFitSlack)) {
		t.Fatalf("one line viewport %d clips line box %d", one.Height-c.TopInset()-c.FrameInsetY(), line)
	}

	three := c.Layout("#00000\n#00000\n#00000", 240, 0)
	if three.FullLines != 3 || three.Scrolls || three.Text != "#00000\n#00000\n#00000" {
		t.Fatalf("three lines: %+v", three)
	}
	if three.Height-c.TopInset()-c.FrameInsetY() < 3*line {
		t.Fatalf("three line viewport %d clips %d", three.Height-c.TopInset()-c.FrameInsetY(), 3*line)
	}
	if three.Height <= one.Height {
		t.Fatalf("three lines not taller than one: %d vs %d", three.Height, one.Height)
	}

	c.MaxHeight = one.Height + line
	over := c.Layout(strings.Repeat("#00000\n", 12)+"#00000", 240, 0)
	if over.FullLines <= 3 || !over.Scrolls || over.Height != c.EffectiveMaxHeight() || over.Truncated {
		t.Fatalf("over max: %+v max %d", over, c.EffectiveMaxHeight())
	}
	if over.Text != strings.Repeat("#00000\n", 12)+"#00000" {
		t.Fatal("over max clipped the string")
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
