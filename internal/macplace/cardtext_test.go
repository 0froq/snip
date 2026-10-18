package macplace

import (
	"testing"

	"github.com/0froq/snip/config"
	"github.com/egoist/mygo/ui"
)

func TestCardMenuOmitsEditWhenSensitive(t *testing.T) {
	titles := CardMenuTitles{Edit: "Edit", Copy: "Copy", Delete: "Delete", Restore: "Restore original"}
	full := CardMenuItems(true, titles)
	if len(full) != 4 || full[0].Action != CardMenuEdit || full[1].Action != CardMenuCopy || full[2].Action != CardMenuDelete || full[3].Action != CardMenuRestore {
		t.Fatalf("menu %+v", full)
	}
	locked := CardMenuItems(false, titles)
	if len(locked) != 3 || locked[0].Action == CardMenuEdit {
		t.Fatalf("sensitive menu %+v", locked)
	}
	for i, item := range locked {
		if item.Action != full[i+1].Action || item.Title != full[i+1].Title {
			t.Fatalf("item %d: %+v", i, item)
		}
	}
}

func TestEditCommitKeys(t *testing.T) {
	if !EditCommit(EditKeyEscape, false) || !EditCommit(EditKeyEscape, true) {
		t.Fatal("escape commits")
	}
	if !EditCommit(EditKeyReturn, true) {
		t.Fatal("command-return commits")
	}
	if EditCommit(EditKeyReturn, false) || EditCommit(EditKeyOther, true) {
		t.Fatal("plain return and other keys stay in the editor")
	}
}

func TestScrollHitLeavesLeftClickToTheCard(t *testing.T) {
	if ScrollHit(false, 1, false) {
		t.Fatal("left mouse down should pass through")
	}
	if !ScrollHit(false, eventScrollWheel, false) || !ScrollHit(false, eventRightMouseDown, false) {
		t.Fatal("scroll and right click stay on the text")
	}
	if !ScrollHit(false, 1, true) {
		t.Fatal("the scroller keeps the left click")
	}
	if !ScrollHit(true, 1, false) {
		t.Fatal("editing takes the left click")
	}
}

func TestTextContentFrameClearsTheResizeBand(t *testing.T) {
	x, y, w, h, ok := TextContentFrame(200, 80, 10, 6, 1, 10, 0)
	if !ok || x != 11 || y != 10 || w != 200-22 || h != 80-20 {
		t.Fatalf("frame %v %v %v %v ok=%v", x, y, w, h, ok)
	}
	if _, _, _, _, ok := TextContentFrame(20, 80, 10, 6, 1, 10, 0); ok {
		t.Fatal("a window narrower than the insets is not a text frame")
	}
}

func TestFontFaceSkipsGenerics(t *testing.T) {
	name, system := FontFace("-apple-system, PingFang SC, Hiragino Sans GB, sans-serif")
	if !system || name != "" {
		t.Fatalf("leading generic is the system font, got %q system %v", name, system)
	}
	if _, system := FontFace("-apple-system, sans-serif"); !system {
		t.Fatal("generics use the system font")
	}
	if name, system := FontFace("  Menlo  "); system || name != "Menlo" {
		t.Fatalf("name %q system %v", name, system)
	}
	if name, system := FontFace("PingFang SC, -apple-system"); system || name != "PingFang SC" {
		t.Fatalf("concrete first %q system %v", name, system)
	}
}

// TestShortCopyFitsTheTextView is the width the card opens at for a short
// line. The fragment inside the pixel frame, title strip, and text-container
// padding has to hold the face the text view draws, or the last word wraps.
func TestShortCopyFitsTheTextView(t *testing.T) {
	const text = "void-jingle deploy"
	cases := []config.Card{
		config.Default().Card,
		func() config.Card {
			c := config.Default().Card
			c.FontFamily = "-apple-system, DejaVu Sans"
			return c
		}(),
		func() config.Card {
			c := config.Default().Card
			c.FontFamily = "DejaVu Sans"
			return c
		}(),
	}
	for _, c := range cases {
		layout := c.Layout(text, 0, 0)
		if layout.FullLines != 1 || layout.Width > c.EffectiveMaxWidth() {
			t.Fatalf("family %q layout %+v", c.FontFamily, layout)
		}
		face := cardTextFont(c)
		ink, _ := ui.ShapeText(text, face, 0).Size()
		_, _, viewW, _, ok := TextContentFrame(
			layout.Width, layout.Height,
			c.EffectivePaddingX(), c.EffectivePaddingY(),
			config.CardBorderPx(), config.CardResizeMargin, config.CardTitleStripPx,
		)
		if !ok {
			t.Fatalf("family %q has no text frame at %d×%d", c.FontFamily, layout.Width, layout.Height)
		}
		fragment := float64(viewW) - 2*config.TextContainerInsetX - 2*config.LineFragmentPadding
		if fragment < float64(ink) {
			t.Fatalf("family %q face %q fragment %v < ink %v (window %d, text view %v)",
				c.FontFamily, face.Family, fragment, ink, layout.Width, viewW)
		}
	}
}

// cardTextFont is the face ApplyFont selects: a concrete family, or the
// configured list when that list starts with the system font.
func cardTextFont(c config.Card) ui.Font {
	family := c.EffectiveFontFamily()
	if name, system := FontFace(family); !system && name != "" {
		family = name
	}
	return ui.Font{Family: family, Size: float32(c.EffectiveFontSize())}
}
