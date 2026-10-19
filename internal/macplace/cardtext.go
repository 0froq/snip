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

// MaskStop is one stop of the clip view's alpha mask. Location runs from
// 0 at the bottom edge to 1 at the top, which is CAGradientLayer's unit
// space. Alpha 1 keeps the text, and 0 lets the card behind it show through.
type MaskStop struct {
	Location float64
	Alpha    float64
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
