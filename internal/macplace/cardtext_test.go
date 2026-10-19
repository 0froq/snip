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

func TestScrollFadesFollowTheEdges(t *testing.T) {
	if top, bottom := ScrollFades(100, 100, 0, fadeEpsilon); top || bottom {
		t.Fatalf("a card that fits has no fade, got %v %v", top, bottom)
	}
	if top, bottom := ScrollFades(200, 80, 0, fadeEpsilon); top || !bottom {
		t.Fatalf("at the top only the bottom fades, got %v %v", top, bottom)
	}
	if top, bottom := ScrollFades(200, 80, 40, fadeEpsilon); !top || !bottom {
		t.Fatalf("mid-scroll fades both edges, got %v %v", top, bottom)
	}
	if top, bottom := ScrollFades(200, 80, 120, fadeEpsilon); !top || bottom {
		t.Fatalf("at the bottom only the top fades, got %v %v", top, bottom)
	}
	if top, bottom := ScrollFades(200, 80, 0.4, fadeEpsilon); top || !bottom {
		t.Fatalf("a fraction of a point at the top stays hidden, got %v %v", top, bottom)
	}
	if top, bottom := ScrollFades(0, 80, 0, fadeEpsilon); top || bottom {
		t.Fatalf("empty content has no fade, got %v %v", top, bottom)
	}
}

func TestFadeBandStaysInsideTheView(t *testing.T) {
	if band := FadeBand(400, FadeHeightPt); band != FadeHeightPt {
		t.Fatalf("band %v", band)
	}
	if band := FadeBand(30, FadeHeightPt); band != 15 {
		t.Fatalf("band %v", band)
	}
	if band := FadeBand(0, FadeHeightPt); band != 0 {
		t.Fatalf("band %v", band)
	}
	if FadeHeightPt < 16 || FadeHeightPt > 24 {
		t.Fatalf("fade height %v", FadeHeightPt)
	}
}

func TestMaskFrameDropsTheScrolledOrigin(t *testing.T) {
	x, y, w, h := MaskFrame(0, 180, 420, 200)
	if x != 0 || y != 0 || w != 420 || h != 200 {
		t.Fatalf("frame %v %v %v %v", x, y, w, h)
	}
	x, y, w, h = MaskFrame(-40, -10, -1, 30)
	if x != 0 || y != 0 || w != 0 || h != 30 {
		t.Fatalf("frame %v %v %v %v", x, y, w, h)
	}
}

func TestViewportFadeHandsTheLayerTheEndStops(t *testing.T) {
	_, startY, _, endY := GradientAxis()
	topLoc := LocationAlongAxis(startY, endY, visualTopY)
	bottomLoc := LocationAlongAxis(startY, endY, visualBottomY)
	if topLoc != 1 || bottomLoc != 0 {
		t.Fatalf("visual top location %v, visual bottom location %v", topLoc, bottomLoc)
	}
	// The alpha at the visual top is the CGColor alpha paired with that location.
	const (
		docH = 500.0
		visH = 200.0
		maxY = docH - visH
	)

	// offset 0: the visual top is opaque, the visual bottom fades.
	locs, alphas := ViewportFade(0, docH, 0, visH, visH, FadeHeightPt, fadeEpsilon, true)
	if alphaAt(locs, alphas, topLoc) != 1 || alphaAt(locs, alphas, bottomLoc) != 0 {
		t.Fatalf("at the top, visual top alpha %v visual bottom alpha %v (%v %v)", alphaAt(locs, alphas, topLoc), alphaAt(locs, alphas, bottomLoc), locs, alphas)
	}
	if !fadeEdges(locs, alphas, 0, 1) {
		t.Fatalf("at the top %+v %+v", locs, alphas)
	}

	// A resting origin of 12 (content inset) is still the top.
	locs, alphas = ViewportFade(12, docH, 12, visH, visH, FadeHeightPt, fadeEpsilon, true)
	if !fadeEdges(locs, alphas, 0, 1) {
		t.Fatalf("inset top %+v %+v", locs, alphas)
	}

	// offset = max: bottom is opaque, top fades.
	locs, alphas = ViewportFade(0, docH, maxY, visH, visH, FadeHeightPt, fadeEpsilon, true)
	if !fadeEdges(locs, alphas, 1, 0) {
		t.Fatalf("at the bottom %+v %+v", locs, alphas)
	}

	// A fraction of a point short of the end still counts as the end.
	locs, alphas = ViewportFade(0, docH, maxY-0.4, visH, visH, FadeHeightPt, fadeEpsilon, true)
	if !fadeEdges(locs, alphas, 1, 0) {
		t.Fatalf("near the bottom %+v %+v", locs, alphas)
	}

	// Past the 1pt slop, the bottom still fades.
	locs, alphas = ViewportFade(0, docH, maxY-1.5, visH, visH, FadeHeightPt, fadeEpsilon, true)
	if !fadeEdges(locs, alphas, 0, 0) {
		t.Fatalf("short of the bottom %+v %+v", locs, alphas)
	}

	// Middle: both edges are clear.
	locs, alphas = ViewportFade(0, docH, 150, visH, visH, FadeHeightPt, fadeEpsilon, true)
	if !fadeEdges(locs, alphas, 0, 0) {
		t.Fatalf("middle %+v %+v", locs, alphas)
	}

	// Non-flipped: y is the bottom of the visible rect. Origin 0 is the
	// bottom of the document, so the bottom stop is opaque.
	locs, alphas = ViewportFade(0, docH, 0, visH, visH, FadeHeightPt, fadeEpsilon, false)
	if !fadeEdges(locs, alphas, 1, 0) {
		t.Fatalf("non-flipped bottom %+v %+v", locs, alphas)
	}
	locs, alphas = ViewportFade(0, docH, maxY, visH, visH, FadeHeightPt, fadeEpsilon, false)
	if !fadeEdges(locs, alphas, 0, 1) {
		t.Fatalf("non-flipped top %+v %+v", locs, alphas)
	}

	if locs, alphas := ViewportFade(0, 100, 0, 200, 200, FadeHeightPt, fadeEpsilon, true); locs != nil || alphas != nil {
		t.Fatalf("content that fits %+v %+v", locs, alphas)
	}
}

func alphaAt(locations, alphas []float64, location float64) float64 {
	for i, loc := range locations {
		if loc == location {
			return alphas[i]
		}
	}
	return -1
}

// fadeEdges reports whether locations run from the visual bottom (0) to the
// visual top (1) and the colors at those ends have the given alphas.
func fadeEdges(locations, alphas []float64, bottomAlpha, topAlpha float64) bool {
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

func TestFadeRectKeepsTheTopStopInsideTheViewport(t *testing.T) {
	x, y, w, h := FadeRect(300, 200, 0, 0, 300, 200)
	if x != 0 || y != 0 || w != 300 || h != 200 {
		t.Fatalf("viewport %v %v %v %v", x, y, w, h)
	}
	// Document height, bottom-aligned: the clear top stop would sit above the mask.
	x, y, w, h = FadeRect(300, 200, 0, 0, 300, 2000)
	if x != 0 || y != 0 || w != 300 || h != 200 {
		t.Fatalf("tall clip %v %v %v %v", x, y, w, h)
	}
	// Shifted up by a scroll offset: the top band leaves and the bottom band moves.
	x, y, w, h = FadeRect(300, 200, 0, 80, 300, 200)
	if x != 0 || y != 0 || w != 300 || h != 200 {
		t.Fatalf("shifted clip %v %v %v %v", x, y, w, h)
	}
}

func TestFadeCoversEaseWithTheScroll(t *testing.T) {
	if FadeOpacitySeconds < 0.15 || FadeOpacitySeconds > 0.20 {
		t.Fatalf("duration %v", FadeOpacitySeconds)
	}
	if FadeOpacityTiming != "easeInEaseOut" {
		t.Fatalf("timing %s", FadeOpacityTiming)
	}
	const (
		docH = 500.0
		visH = 200.0
		maxY = docH - visH
	)
	under := FadeUnderlay(visH, FadeHeightPt)
	if len(under) != 4 || under[0] != (MaskStop{0, 0}) || under[3] != (MaskStop{1, 0}) {
		t.Fatalf("underlay %+v", under)
	}
	assert := func(offset, topOpacity, bottomOpacity float64) {
		t.Helper()
		top, bottom, ok := ScrollFadeCovers(docH, visH, offset, fadeEpsilon, 0, 0, 300, visH, FadeHeightPt)
		if !ok {
			t.Fatalf("offset %v has no covers", offset)
		}
		if top.Opacity != topOpacity || bottom.Opacity != bottomOpacity {
			t.Fatalf("offset %v cover opacity top %v bottom %v", offset, top.Opacity, bottom.Opacity)
		}
		if top.Y != visH-FadeHeightPt || top.H != FadeHeightPt || bottom.Y != 0 || bottom.H != FadeHeightPt {
			t.Fatalf("offset %v frames top %+v bottom %+v", offset, top, bottom)
		}
		if top.Y+top.H != visH || bottom.Y+bottom.H != FadeHeightPt {
			t.Fatalf("offset %v bands leave the viewport %+v %+v", offset, top, bottom)
		}
		locs, alphas := ViewportFade(0, docH, offset, visH, visH, FadeHeightPt, fadeEpsilon, true)
		if alphaAt(locs, alphas, 1) != top.Opacity || alphaAt(locs, alphas, 0) != bottom.Opacity {
			t.Fatalf("offset %v baked %+v %+v covers %v %v", offset, locs, alphas, top.Opacity, bottom.Opacity)
		}
	}
	// At the top the top cover is solid (fade hidden) and the bottom cover
	// is clear (fade shown). Leaving either edge clears that cover.
	assert(0, 1, 0)
	assert(0.4, 1, 0)
	assert(150, 0, 0)
	assert(maxY, 0, 1)
	assert(maxY-0.4, 0, 1)
	assert(maxY-1.5, 0, 0)

	// Content that fits still has covers, both solid, so a later overflow
	// can ease a fade in instead of attaching a mask already at full strength.
	top, bottom, ok := ScrollFadeCovers(100, 200, 0, fadeEpsilon, 4, 6, 300, 200, FadeHeightPt)
	if !ok || top.Opacity != 1 || bottom.Opacity != 1 {
		t.Fatalf("content that fits %+v %+v ok=%v", top, bottom, ok)
	}
	if top.Y != 200-FadeHeightPt+6 || bottom.Y != 6 {
		t.Fatalf("shifted bands %+v %+v", top, bottom)
	}
	if _, _, ok := ScrollFadeCovers(500, 200, 100, fadeEpsilon, 0, 0, 0, 200, FadeHeightPt); ok {
		t.Fatal("a zero-width gradient has no covers")
	}
	if documentOffset(12, docH, 12, visH, true) != 0 {
		t.Fatal("a resting inset is the top")
	}
	if documentOffset(0, docH, 0, visH, false) != maxY || documentOffset(0, docH, maxY, visH, false) != 0 {
		t.Fatal("non-flipped offset is measured from the top")
	}
}

func TestMaskStopsClearOnlyTheScrollingEdge(t *testing.T) {
	if stops := MaskStops(200, 20, false, false); stops != nil {
		t.Fatalf("a card that fits has no mask, got %+v", stops)
	}
	if stops := MaskStops(0, 20, true, true); stops != nil {
		t.Fatalf("empty view %+v", stops)
	}
	bottom := MaskStops(200, 20, false, true)
	if len(bottom) != 3 || bottom[0] != (MaskStop{0, 0}) || bottom[1] != (MaskStop{20.0 / 200.0, 1}) || bottom[2] != (MaskStop{1, 1}) {
		t.Fatalf("bottom %+v", bottom)
	}
	top := MaskStops(200, 20, true, false)
	if len(top) != 3 || top[0] != (MaskStop{0, 1}) || top[1] != (MaskStop{1 - 20.0/200.0, 1}) || top[2] != (MaskStop{1, 0}) {
		t.Fatalf("top %+v", top)
	}
	both := MaskStops(200, 20, true, true)
	if len(both) != 4 || both[0].Alpha != 0 || both[1] != (MaskStop{20.0 / 200.0, 1}) || both[2] != (MaskStop{1 - 20.0/200.0, 1}) || both[3].Alpha != 0 {
		t.Fatalf("both %+v", both)
	}
	// A band taller than the view still meets in the middle, clear at both edges.
	tight := MaskStops(30, 40, true, true)
	if len(tight) != 4 || tight[0].Alpha != 0 || tight[1].Location != 0.5 || tight[3].Alpha != 0 {
		t.Fatalf("tight %+v", tight)
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
