package macplace

import (
	"math"
	"strings"
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

func TestUpdateFadeFollowsTheClip(t *testing.T) {
	const (
		docH  = 500.0
		clipH = 200.0
		line  = 18.0
		maxY  = docH - clipH
	)
	// A previous offset must not stick. The function has no memory.
	_ = updateFade(150, clipH, docH, line)
	atTop := updateFade(0, clipH, docH, line)
	if !fadeEdges(atTop, 0, 1) || atTop.Band != line/2 {
		t.Fatalf("at the top %+v", atTop)
	}
	if atTop.Locations[1]*clipH > line*0.5+1e-9 {
		t.Fatalf("band covers a line %+v", atTop)
	}
	// Half a point is still the top.
	if mask := updateFade(fadeEdgeSlop, clipH, docH, line); !fadeEdges(mask, 0, 1) {
		t.Fatalf("resting top %+v", mask)
	}
	if mask := updateFade(fadeEdgeSlop+0.01, clipH, docH, line); !fadeEdges(mask, 0, 0) {
		t.Fatalf("just off the top %+v", mask)
	}

	if mask := updateFade(150, clipH, docH, line); !fadeEdges(mask, 0, 0) {
		t.Fatalf("middle %+v", mask)
	}
	if mask := updateFade(maxY, clipH, docH, line); !fadeEdges(mask, 1, 0) {
		t.Fatalf("at the bottom %+v", mask)
	}
	// The far edge within half a point of the document is the bottom.
	if mask := updateFade(maxY-fadeEdgeSlop, clipH, docH, line); !fadeEdges(mask, 1, 0) {
		t.Fatalf("near the bottom %+v", mask)
	}
	if mask := updateFade(maxY-fadeEdgeSlop-0.01, clipH, docH, line); !fadeEdges(mask, 0, 0) {
		t.Fatalf("short of the bottom %+v", mask)
	}

	if mask := updateFade(40, clipH, 180, line); mask.Locations != nil || mask.Band != 0 {
		t.Fatalf("content that fits %+v", mask)
	}
	if mask := updateFade(0, clipH, clipH, line); mask.Locations != nil {
		t.Fatalf("exact fit %+v", mask)
	}
	if mask := updateFade(0, 0, docH, line); mask.Locations != nil {
		t.Fatalf("empty clip %+v", mask)
	}
	if mask := updateFade(0, clipH, docH, 0); mask.Locations != nil || mask.Band != 0 {
		t.Fatalf("unknown line %+v", mask)
	}
	// A short viewport caps the band at half the clip.
	if mask := updateFade(0, 10, docH, line); mask.Band != 5 {
		t.Fatalf("capped band %+v", mask)
	}
}

func TestClampClipOriginWhenContentFits(t *testing.T) {
	// Scrolled down in a tall card. After the resize the text fits, so the
	// origin has to come back to 0. Leaving it above 0 clips the first line
	// above the top inset.
	if got := clampClipOrigin(80, 120, 200); got != 0 {
		t.Fatalf("fitted content origin %v", got)
	}
	if got := clampClipOrigin(0, 120, 200); got != 0 {
		t.Fatalf("already at the top %v", got)
	}
	if got := clampClipOrigin(-12, 500, 200); got != 0 {
		t.Fatalf("above the top %v", got)
	}
	if got := clampClipOrigin(40, 500, 200); got != 40 {
		t.Fatalf("in range %v", got)
	}
	if got := clampClipOrigin(400, 500, 200); got != 300 {
		t.Fatalf("past the end %v", got)
	}
}

func fadeEdges(mask FadeMask, bottomAlpha, topAlpha float64) bool {
	locations, alphas := mask.Locations, mask.Alphas
	n := len(locations)
	if n < 2 || n != len(alphas) || locations[0] != 0 || locations[n-1] != 1 {
		return false
	}
	if alphas[0] != bottomAlpha || alphas[n-1] != topAlpha {
		return false
	}
	for i := 1; i < n; i++ {
		if locations[i] <= locations[i-1] {
			return false
		}
	}
	return true
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
		_, _, viewW, viewH, ok := TextContentFrame(
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
		_, shapedH := ui.ShapeText(text, face, float32(fragment)).Size()
		need := math.Ceil(float64(shapedH)) + 2*config.TextContainerInsetY
		if layout.Scrolls || viewH < need {
			t.Fatalf("family %q view height %v need %v scrolls %v", c.FontFamily, viewH, need, layout.Scrolls)
		}
	}
}

func TestMultiLineViewportFitsWithoutScrolling(t *testing.T) {
	c := config.Default().Card
	one := c.Layout("void-jingle deploy", 0, 0)
	multi := c.Layout("void-jingle deploy\nagain\nand more", 0, 0)
	if one.Scrolls || multi.Scrolls || multi.FullLines != 3 || multi.Height <= one.Height {
		t.Fatalf("one %+v multi %+v", one, multi)
	}
	_, _, _, viewH, ok := TextContentFrame(
		multi.Width, multi.Height,
		c.EffectivePaddingX(), c.EffectivePaddingY(),
		config.CardBorderPx(), config.CardResizeMargin, config.CardTitleStripPx,
	)
	if !ok {
		t.Fatal("no text frame")
	}
	if viewH < float64(3*c.LineBoxPx()) {
		t.Fatalf("view height %v for 3 lines of %d", viewH, c.LineBoxPx())
	}
	c.MaxHeight = one.Height
	over := c.Layout(strings.Repeat("void-jingle deploy\n", 12)+"end", 0, 0)
	if !over.Scrolls || over.Height != c.EffectiveMaxHeight() {
		t.Fatalf("over max %+v", over)
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
