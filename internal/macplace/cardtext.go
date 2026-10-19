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

// fadeEdgeSlop is how far the clip may sit from an edge before that edge fades.
// Half a point absorbs a fractional scroll offset without keeping a fade on
// at the resting top or bottom.
const fadeEdgeSlop = 0.5

// FadeMask is the alpha mask for one clip view. Locations is nil when the
// text needs no mask. Location 0 is the visual bottom and location 1 is the
// visual top: the scroll view's mask is composited with unit y=0 at the top,
// so the gradient axis runs from (0.5, 1) to (0.5, 0). Alpha 1 keeps the
// text. Band is the fade height in points, at most half a line.
type FadeMask struct {
	Band      float64
	Locations []float64
	Alphas    []float64
}

// updateFade is the mask for a clip view. clipY is the clip bounds origin
// (the scroll offset from the top), clipH the viewport, docH the document
// height, and lineHeight one line. The same inputs always give the same
// mask: nothing is remembered from an earlier scroll, resize, or edit.
// Text that fits has no fade. The top fades only once the clip has left
// the top, and the bottom fades only while the clip's far edge is short of
// the document. The band is half a line, and at most half the viewport.
func updateFade(clipY, clipH, docH, lineHeight float64) FadeMask {
	if clipH <= 0 || docH <= clipH {
		return FadeMask{}
	}
	top := clipY > fadeEdgeSlop
	bottom := clipY+clipH < docH-fadeEdgeSlop
	if !top && !bottom {
		return FadeMask{}
	}
	band := 0.5 * lineHeight
	if band > 0.5*clipH {
		band = 0.5 * clipH
	}
	if band <= 0 {
		return FadeMask{}
	}
	frac := band / clipH
	var locations, alphas []float64
	switch {
	case top && bottom:
		if frac >= 0.5 {
			locations = []float64{0, 0.5, 1}
			alphas = []float64{0, 1, 0}
		} else {
			locations = []float64{0, frac, 1 - frac, 1}
			alphas = []float64{0, 1, 1, 0}
		}
	case bottom:
		locations = []float64{0, frac, 1}
		alphas = []float64{0, 1, 1}
	case top:
		locations = []float64{0, 1 - frac, 1}
		alphas = []float64{1, 1, 0}
	default:
		return FadeMask{}
	}
	return FadeMask{Band: band, Locations: locations, Alphas: alphas}
}

// clampClipOrigin keeps a clip-view bounds origin inside the document.
// The range is [0, max(0, docH-clipH)]. Content that fits, or a clip
// already at the top, stays at 0, so a resize cannot leave the first line
// above the top inset.
func clampClipOrigin(origin, docH, clipH float64) float64 {
	if origin < 0 {
		origin = 0
	}
	maxOrigin := docH - clipH
	if maxOrigin < 0 {
		maxOrigin = 0
	}
	if origin > maxOrigin {
		return maxOrigin
	}
	return origin
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
