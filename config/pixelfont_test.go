package config

import (
	"strings"
	"testing"

	"git.zacharyzhang.com/ZacharyZhang-NY/8bitMyGoUI/pixel"
)

func TestCardFontDefaultsToSystem(t *testing.T) {
	d := Default()
	if d.Card.EffectiveCardFont() != CardFontSystem || d.Card.PixelScale() != 1 {
		t.Fatalf("font %q scale %d", d.Card.EffectiveCardFont(), d.Card.PixelScale())
	}
	d.Card.FontKind = "nope"
	if d.Card.EffectiveCardFont() != CardFontSystem {
		t.Fatalf("unknown %q", d.Card.EffectiveCardFont())
	}
	f := FormFrom(Default())
	f.CardFont = "pixel"
	got, err := f.Parse(Default())
	if err != nil || !got.Card.PixelFont() || got.Card.FontKind != CardFontPixel {
		t.Fatalf("parse %v %+v", err, got.Card.FontKind)
	}
	f.CardFont = "system"
	got, err = f.Parse(got)
	if err != nil || got.Card.FontKind != "" || got.Card.PixelFont() {
		t.Fatalf("system %v %q", err, got.Card.FontKind)
	}
	f.CardFont = "comic"
	if _, err := f.Parse(Default()); err == nil {
		t.Fatal("expected card font error")
	}
	if PixelScale(14) != 1 || PixelScale(47) != 1 || PixelScale(48) != 2 || PixelScale(80) != 2 {
		t.Fatalf("scales %d %d %d %d", PixelScale(14), PixelScale(47), PixelScale(48), PixelScale(80))
	}
}

func TestPixelCardMeasuresTheBitmap(t *testing.T) {
	c := Default().Card
	c.FontKind = CardFontPixel
	const text = "void-jingle deploy"
	layout := c.Layout(text, 0, 0)
	unit := c.PixelScale() * PixelDot
	ink := pixel.Measure(text) * unit
	wantW := ink + 2*c.FrameInsetX()
	if layout.FullLines != 1 || layout.Scrolls || layout.Width != wantW {
		t.Fatalf("layout %+v want width %d", layout, wantW)
	}
	line := pixel.LineHeight * unit
	block := c.textBlockPx(1, float32(line))
	if block < line || layout.Height != c.windowHeight(block) {
		t.Fatalf("height %d block %d line %d", layout.Height, block, line)
	}
	wide := c
	wide.FontSize = 64
	if wide.PixelScale() != 2 {
		t.Fatalf("scale %d", wide.PixelScale())
	}
	big := wide.Layout(text, 0, 0)
	if big.Height <= layout.Height || big.Width <= layout.Width {
		t.Fatalf("larger face %+v vs %+v", big, layout)
	}
	narrow := c.FrameInsetX()*2 + unit*pixel.Measure("A")
	wrapped := c.Layout("AAA", narrow, 0)
	if wrapped.FullLines < 2 {
		t.Fatalf("wrapped %+v dots %d", wrapped, c.pixelWrapDots(narrow))
	}
	if strings.Join(pixel.Lines("AAA", c.pixelWrapDots(narrow)), "|") == "AAA" {
		t.Fatal("measure and wrap disagree")
	}
}
