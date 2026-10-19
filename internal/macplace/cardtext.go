package macplace

import (
	"fmt"

	"github.com/0froq/snip/config"
)

// CardMenuAction is one item in a card's context menu.
type CardMenuAction int

const (
	CardMenuEdit CardMenuAction = iota + 1
	CardMenuCopy
	CardMenuDelete
	CardMenuRestore
)

// CardMenuTitles are the labels shown in the context menu.
type CardMenuTitles struct {
	Edit    string
	Copy    string
	Delete  string
	Restore string
}

// CardMenuItem is one row of the context menu.
type CardMenuItem struct {
	Action CardMenuAction
	Title  string
}

// CardMenuItems is Edit, Copy, Delete, Restore original. Sensitive cards
// omit Edit. The remaining items stay in that order.
func CardMenuItems(canEdit bool, titles CardMenuTitles) []CardMenuItem {
	items := make([]CardMenuItem, 0, 4)
	if canEdit {
		items = append(items, CardMenuItem{Action: CardMenuEdit, Title: titles.Edit})
	}
	items = append(items,
		CardMenuItem{Action: CardMenuCopy, Title: titles.Copy},
		CardMenuItem{Action: CardMenuDelete, Title: titles.Delete},
		CardMenuItem{Action: CardMenuRestore, Title: titles.Restore},
	)
	return items
}

// EditKey is a key the text view turns into a command.
type EditKey int

const (
	EditKeyEscape EditKey = iota
	EditKeyReturn
	EditKeyOther
)

// EditCommit reports whether this key ends an edit and saves it.
// Escape commits. Return commits only with Command held; a plain Return
// inserts a newline. Any other key does not commit.
func EditCommit(key EditKey, command bool) bool {
	switch key {
	case EditKeyEscape:
		return true
	case EditKeyReturn:
		return command
	case EditKeyOther:
		return false
	default:
		panic(fmt.Sprintf("snip: unhandled edit key %d", int(key)))
	}
}

// AppKit event types the scroll view must keep. Left-button events pass
// through so the card can still be dragged and double-clicked.
const (
	eventRightMouseDown    = 3
	eventRightMouseUp      = 4
	eventRightMouseDragged = 5
	eventScrollWheel       = 22
)

// ScrollHit reports whether the scroll view should take the event.
// An edit takes every event. Otherwise the vertical scroller, a right
// click, and a scroll wheel stay with the scroll view, and a left click
// falls through to the card.
func ScrollHit(editing bool, eventType int, inScroller bool) bool {
	if editing || inScroller {
		return true
	}
	switch eventType {
	case eventRightMouseDown, eventRightMouseUp, eventRightMouseDragged, eventScrollWheel:
		return true
	default:
		return false
	}
}

// TextContentFrame is the scroll view's frame inside the window, in the
// content view's bottom-left coordinates. The inset is the card padding
// plus its border, and at least the resize band, so the text does not
// cover the edges that resize the card. topExtra raises the top inset
// when a title strip is taller than that band.
func TextContentFrame(winW, winH, padX, padY, border, resizeMargin, topExtra int) (x, y, w, h float64, ok bool) {
	left, top, right, bottom := config.TextFrameInsets(padX, padY, border, resizeMargin, topExtra)
	if winW <= left+right || winH <= top+bottom || left < 0 || top < 0 || right < 0 || bottom < 0 {
		return 0, 0, 0, 0, false
	}
	return float64(left), float64(bottom), float64(winW - left - right), float64(winH - top - bottom), true
}

func pointInNSRect(px, py, x, y, w, h float64) bool {
	return w > 0 && h > 0 && px >= x && py >= y && px < x+w && py < y+h
}

// FontFace is the face the card text view draws with. It is the same face
// the card measures, so a short line's width and height match the view.
func FontFace(family string) (name string, system bool) {
	return config.ResolveFont(family)
}

const (
	// FadeHeightPt is the scroll-edge gradient, in points.
	FadeHeightPt = 20
	// fadeEpsilon absorbs a fractional scroll offset so the fade does not flicker.
	fadeEpsilon = 1
)

// ScrollFades reports which edges of a vertical scroller should fade.
// contentH is the document height, viewH the visible height, and offset
// the distance from the top of the document. A card that fits has neither
// fade. The top fade appears once the document has moved down, and the
// bottom fade stays until the document's end is in view.
func ScrollFades(contentH, viewH, offset, epsilon float64) (top, bottom bool) {
	if contentH <= 0 || viewH <= 0 {
		return false, false
	}
	if epsilon < 0 {
		epsilon = 0
	}
	if contentH <= viewH+epsilon {
		return false, false
	}
	maxOff := contentH - viewH
	if offset > epsilon {
		top = true
	}
	if offset < maxOff-epsilon {
		bottom = true
	}
	return top, bottom
}

// FadeBand is the gradient height that fits in the visible area. It is
// capped at half the view so the two edges never cross.
func FadeBand(viewH, want float64) float64 {
	if viewH <= 0 || want <= 0 {
		return 0
	}
	half := viewH / 2
	if want > half {
		return half
	}
	return want
}

// MaskFrame is the fade's rectangle in a layer that does not scroll.
// A clip-view bounds origin moves with the document; using it as the mask
// origin carries the fade along. The viewport mask always starts at zero
// and only takes the visible width and height.
func MaskFrame(boundsX, boundsY, width, height float64) (x, y, w, h float64) {
	_, _ = boundsX, boundsY
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	return 0, 0, width, height
}

// MaskStop is one stop of the viewport's alpha mask. Location 0 is the
// start of GradientAxis and location 1 is the end. Alpha 1 keeps the text,
// and 0 lets the card behind it show through.
type MaskStop struct {
	Location float64
	Alpha    float64
}

// The scroll view's layer keeps geometryFlipped off, so a sublayer's frame
// still grows upward. AppKit composites that layer as a mask with unit y=0
// at the top of the scroll view: a gradient that starts at (0.5, 0) draws
// its first stop on the visual top. The axis therefore starts at unit y=1,
// the visual bottom, and ends at unit y=0, the visual top. Location 1 is
// the visual top, which is where the top alpha sits.
const (
	visualBottomY = 1.0
	visualTopY    = 0.0
)

// GradientAxis is the line the viewport mask draws along, from the visual
// bottom to the visual top.
func GradientAxis() (startX, startY, endX, endY float64) {
	return 0.5, visualBottomY, 0.5, visualTopY
}

// LocationAlongAxis is the gradient location of a unit-space y. 0 is the
// start of the axis and 1 is the end.
func LocationAlongAxis(startY, endY, unitY float64) float64 {
	span := endY - startY
	if span == 0 {
		return 0
	}
	return (unitY - startY) / span
}

// ViewportFade is the colors and locations handed to CAGradientLayer.
// docOrigin/docH is the clip view's documentRect, visOrigin/visH its
// documentVisibleRect, both in the document view's coordinates. maskH is
// the gradient's pixel height. flipped is the document view: a flipped
// NSTextView measures y from the top, so the top of the document is
// docOrigin, not zero. A content inset that leaves the resting origin at
// 12 is still the top. The bottom is the visible rect's far edge within
// fadeEpsilon of the document's far edge, so a fraction of a point still
// counts as the end. Location 1 is the visual top (GradientAxis's end).
// The alpha at a location is the CGColor alpha.
// Empty slices mean the layer gets no mask.
// documentOffset is how far the visible rect sits from the top of the
// document. A flipped document view measures y from the top. A non-flipped
// view's origin is the bottom edge, so the top is the space still above
// the visible rect.
func documentOffset(docOrigin, docH, visOrigin, visH float64, flipped bool) float64 {
	if !flipped {
		return (docOrigin + docH) - (visOrigin + visH)
	}
	return visOrigin - docOrigin
}

func ViewportFade(docOrigin, docH, visOrigin, visH, maskH, fadeH, epsilon float64, flipped bool) (locations, alphas []float64) {
	if maskH <= 0 {
		maskH = visH
	}
	offset := documentOffset(docOrigin, docH, visOrigin, visH, flipped)
	top, bottom := ScrollFades(docH, visH, offset, epsilon)
	stops := MaskStops(maskH, FadeBand(maskH, fadeH), top, bottom)
	if len(stops) == 0 {
		return nil, nil
	}
	locations = make([]float64, len(stops))
	alphas = make([]float64, len(stops))
	for i, stop := range stops {
		locations[i] = stop.Location
		alphas[i] = stop.Alpha
	}
	return locations, alphas
}

// FadeRect is the gradient's frame inside the scroll-view mask. It is the
// whole scroll view, not the clip view. An overlay scroller does not inset
// the clip, so a gradient only as wide as a gutter-clipped frame leaves the
// last glyphs outside the fade. A document-tall or y-shifted clip frame
// would also park the clear top stop above the mask.
func FadeRect(scrollW, scrollH, clipX, clipY, clipW, clipH float64) (x, y, w, h float64) {
	_, _, _, _ = clipX, clipY, clipW, clipH
	if scrollW < 0 {
		scrollW = 0
	}
	if scrollH < 0 {
		scrollH = 0
	}
	return 0, 0, scrollW, scrollH
}

// KnobFrame is the scroller knob in the scroll view's coordinates. The
// solid mask patch is this rectangle and nothing else, so the fade still
// covers the text under the track. A hidden scroller or an empty knob
// returns ok false.
// knobX/knobY are in the scroller's bounds, with the bounds origin removed.
// scrollFlipped and scrollerFlipped are the two views' isFlipped.
func KnobFrame(visible, scrollFlipped, scrollerFlipped bool, scrollerX, scrollerY, scrollerH, knobX, knobY, knobW, knobH float64) (x, y, w, h float64, ok bool) {
	if !visible || scrollerH <= 0 || knobW <= 0 || knobH <= 0 {
		return 0, 0, 0, 0, false
	}
	x = scrollerX + knobX
	if scrollFlipped == scrollerFlipped {
		y = scrollerY + knobY
	} else {
		y = scrollerY + (scrollerH - knobY - knobH)
	}
	return x, y, knobW, knobH, true
}

// MaskStops is the vertical alpha mask for a viewport viewH points tall.
// fadeH is the band at an edge. top and bottom are the edges that still
// have more text past the viewport. No edge returns no stops, so the
// clip view keeps no mask. Each fading edge goes from opaque on the
// inside of the band to clear at the viewport edge.
func MaskStops(viewH, fadeH float64, top, bottom bool) []MaskStop {
	if viewH <= 0 || (!top && !bottom) {
		return nil
	}
	if fadeH < 0 {
		fadeH = 0
	}
	if fadeH > viewH/2 {
		fadeH = viewH / 2
	}
	if fadeH <= 0 {
		return nil
	}
	band := fadeH / viewH
	if top && bottom {
		return []MaskStop{{0, 0}, {band, 1}, {1 - band, 1}, {1, 0}}
	}
	if bottom {
		return []MaskStop{{0, 0}, {band, 1}, {1, 1}}
	}
	return []MaskStop{{0, 1}, {1 - band, 1}, {1, 0}}
}

// FadeOpacitySeconds is how long an edge fade takes to appear or disappear.
// FadeOpacityTiming is the Core Animation curve name for both directions.
// FadeContainerFlipped is written to the mask container's geometryFlipped.
// It stays off so the scroller frame and the gradient axis keep their
// current space. In that space the mask is composited with the container's
// low y on the visual top, and cover frames are computed to match.
const (
	FadeOpacitySeconds   = 0.18
	FadeOpacityTiming    = "easeInEaseOut"
	FadeContainerFlipped = false
)

// FadeCover is the solid layer over one end of the viewport gradient.
// Opacity 0 shows that end's fade. Opacity 1 hides it, so the text stays
// solid. Animating this opacity is what eases the fade in and out.
type FadeCover struct {
	X, Y, W, H float64
	Opacity    float64
}

// FadeUnderlay is the gradient under the covers. Both edges stay clear so
// changing a cover's opacity reveals one edge without rebuilding the stops.
func FadeUnderlay(viewH, fadeH float64) []MaskStop {
	return MaskStops(viewH, FadeBand(viewH, fadeH), true, true)
}

func fadeCoverOpacity(edgeFades bool) float64 {
	if edgeFades {
		return 0
	}
	return 1
}

// FadeCovers are the solid covers for the visual top and visual bottom.
// geometryFlipped is the mask container's geometryFlipped. With it off, y
// grows upward and AppKit still composites the mask so the low edge is the
// visual top: the top cover is that low band and the bottom cover is the
// high band. With it on, the container's own flip cancels that composite,
// so the low edge is the visual bottom and the bands swap. A viewport too
// short for a band has none.
func FadeCovers(gradX, gradY, gradW, gradH, fadeH float64, topFades, bottomFades, geometryFlipped bool) (top, bottom FadeCover, ok bool) {
	band := FadeBand(gradH, fadeH)
	if gradW <= 0 || band <= 0 {
		return FadeCover{}, FadeCover{}, false
	}
	topY := gradY
	bottomY := gradY + gradH - band
	if geometryFlipped {
		topY, bottomY = bottomY, topY
	}
	top = FadeCover{
		X: gradX, Y: topY, W: gradW, H: band,
		Opacity: fadeCoverOpacity(topFades),
	}
	bottom = FadeCover{
		X: gradX, Y: bottomY, W: gradW, H: band,
		Opacity: fadeCoverOpacity(bottomFades),
	}
	return top, bottom, true
}

// ScrollFadeCovers is the cover pair for a distance from the top of the
// document. The gradient under them always clears both edges.
func ScrollFadeCovers(contentH, viewH, offset, epsilon, gradX, gradY, gradW, gradH, fadeH float64, geometryFlipped bool) (top, bottom FadeCover, ok bool) {
	topOn, bottomOn := ScrollFades(contentH, viewH, offset, epsilon)
	return FadeCovers(gradX, gradY, gradW, gradH, fadeH, topOn, bottomOn, geometryFlipped)
}

// CardTextOptions describes the scroll view embedded in a card window.
type CardTextOptions struct {
	Text               string
	FontFamily         string
	FontSize           float64
	Red, Green, Blue   uint8
	CanEdit            bool
	Titles             CardMenuTitles
	Width, Height      int
	PadX, PadY, Border int
	ResizeMargin       int
	TopExtra           int
	OnAction           func(CardMenuAction)
	OnCommit           func(string)
}
