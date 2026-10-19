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
func ViewportFade(docOrigin, docH, visOrigin, visH, maskH, fadeH, epsilon float64, flipped bool) (locations, alphas []float64) {
	if maskH <= 0 {
		maskH = visH
	}
	offset := visOrigin - docOrigin
	if !flipped {
		// Non-flipped: the rect origin is the bottom edge. The distance
		// from the top is whatever is still above the visible rect.
		offset = (docOrigin + docH) - (visOrigin + visH)
	}
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

// FadeRect is the gradient's frame inside an unflipped scroll-view mask.
// The clip view's frame is in that same space only while it sits inside
// the scroll view. A taller frame (the document height) or one shifted up
// by a scroll offset draws the clear top stop above the mask, and the
// bottom stop still lands on the bottom edge.
func FadeRect(scrollW, scrollH, clipX, clipY, clipW, clipH float64) (x, y, w, h float64) {
	if scrollW < 0 {
		scrollW = 0
	}
	if scrollH < 0 {
		scrollH = 0
	}
	if clipW <= 0 || clipH <= 0 || clipX < 0 || clipY < 0 ||
		clipX+clipW > scrollW+0.5 || clipY+clipH > scrollH+0.5 {
		return 0, 0, scrollW, scrollH
	}
	return clipX, clipY, clipW, clipH
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
