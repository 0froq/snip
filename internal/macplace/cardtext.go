package macplace

import "fmt"

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
// cover the edges that resize the card.
func TextContentFrame(winW, winH, padX, padY, border, resizeMargin int) (x, y, w, h float64, ok bool) {
	insetX := padX + border
	insetY := padY + border
	if resizeMargin > insetX {
		insetX = resizeMargin
	}
	if resizeMargin > insetY {
		insetY = resizeMargin
	}
	if winW <= 2*insetX || winH <= 2*insetY || insetX < 0 || insetY < 0 {
		return 0, 0, 0, 0, false
	}
	return float64(insetX), float64(insetY), float64(winW - 2*insetX), float64(winH - 2*insetY), true
}

func pointInNSRect(px, py, x, y, w, h float64) bool {
	return w > 0 && h > 0 && px >= x && py >= y && px < x+w && py < y+h
}

// FontFace is the first concrete family in a CSS-style list. Generic
// names (-apple-system, system-ui, sans-serif, serif, monospace) are
// skipped. system is true when the list has no concrete family.
func FontFace(family string) (name string, system bool) {
	start := 0
	for i := 0; i <= len(family); i++ {
		if i != len(family) && family[i] != ',' {
			continue
		}
		part := trimSpace(family[start:i])
		start = i + 1
		if part == "" {
			continue
		}
		if genericFont(part) {
			system = true
			continue
		}
		return part, false
	}
	return "", true
}

func genericFont(name string) bool {
	switch lowerASCII(name) {
	case "-apple-system", "system-ui", "sans-serif", "serif", "monospace":
		return true
	default:
		return false
	}
}

func lowerASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}

func trimSpace(s string) string {
	i := 0
	j := len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t') {
		j--
	}
	return s[i:j]
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
	OnAction           func(CardMenuAction)
	OnCommit           func(string)
}
