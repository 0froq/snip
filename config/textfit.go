package config

import (
	"math"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

const cardBorderPx = 1

// CardResizeMargin is the drag band around a card. The native text view is
// inset by at least this much so the band stays hittable, and the window
// has to be tall enough for the text plus that inset.
const CardResizeMargin = 10

// TextContainerInsetX and TextContainerInsetY are NSTextView's
// textContainerInset, in points. LineFragmentPadding is NSTextContainer's
// lineFragmentPadding. The card sets these on the text view and adds them
// to the measured size, so the glyphs are not clipped by the defaults
// (inset 0, padding 5).
const (
	TextContainerInsetX = 0.0
	TextContainerInsetY = 0.0
	LineFragmentPadding = 5.0
)

// CardBorderPx is the slip border width in the card view.
func CardBorderPx() int { return cardBorderPx }

// FrameInsetX is how far the text view sits from the left and right edges.
func (c Card) FrameInsetX() int { return frameInset(c.EffectivePaddingX()) }

// FrameInsetY is how far the text view sits from the top and bottom edges.
func (c Card) FrameInsetY() int { return frameInset(c.EffectivePaddingY()) }

func frameInset(pad int) int {
	inset := pad + cardBorderPx
	if CardResizeMargin > inset {
		return CardResizeMargin
	}
	return inset
}

// LineBoxPx is one line's height: ascent, descent, and leading, rounded up
// the way a line fragment is.
func (c Card) LineBoxPx() int {
	h := int(math.Ceil(float64(c.lineBox())))
	if h < 1 {
		return 1
	}
	return h
}

func (c Card) cardFont() ui.Font {
	return c.UIFont(c.EffectiveFontSize())
}

// UIFont is the face used for cards, badges, which-key text and for measuring
// lines. size <= 0 uses the configured font size.
func (c Card) UIFont(size int) ui.Font {
	if size <= 0 {
		size = c.EffectiveFontSize()
	}
	return ui.Font{Family: c.EffectiveFontFamily(), Size: float32(size)}
}

func (c Card) textWidthForSlip(slipWidth int) float32 {
	// The scroll view is inside the frame inset. NSTextView then eats
	// textContainerInset and lineFragmentPadding before the glyphs wrap.
	inner := float64(slipWidth) - 2*float64(c.FrameInsetX()) - 2*TextContainerInsetX - 2*LineFragmentPadding
	if inner < 1 {
		return 1
	}
	return float32(inner)
}

func (c Card) lineBox() float32 {
	m := c.cardFont().Metrics()
	h := m.Ascent + m.Descent + m.LineGap
	if h < 1 {
		return float32(c.EffectiveLineHeight())
	}
	return h
}

// CardLayout is the exact text and geometry drawn inside a card.
type CardLayout struct {
	Text         string
	Width        int
	Height       int
	VisibleLines int
	FullLines    int
	Truncated    bool
	// Scrolls is true when the text is taller than the window. The card
	// shows it in a scroll view instead of clipping it.
	Scrolls bool
}

// linesForText counts the visual lines MyGo actually lays out, by distinct
// caret rows. Dividing the height by the Latin line box overcounts lines
// that use a taller CJK fallback font.
func (c Card) linesForText(text string, width int) int {
	if text == "" {
		return 1
	}
	tl := ui.ShapeText(text, c.cardFont(), c.textWidthForSlip(width))
	// Carets on one line can sit at different heights when Latin and CJK
	// fallback glyphs mix, so cluster caret centers within half a line box.
	n := len(text)
	ys := make([]float64, 0, n+1)
	for i := 0; i <= n; i++ {
		r := tl.Caret(i)
		ys = append(ys, float64(r.Y+r.H/2))
	}
	sort.Float64s(ys)
	gap := float64(c.lineBox()) / 2
	rows := 1
	start := ys[0]
	for _, y := range ys[1:] {
		if y-start > gap {
			rows++
			start = y
		}
	}
	return rows
}

// measureBox is the full text's line count and the window height that
// shows every line, using the same font the card renders with.
func (c Card) measureBox(display string, width int) (lines, height int) {
	if strings.TrimSpace(display) == "" {
		display = " "
	}
	lines = c.linesForText(display, width)
	if lines < 1 {
		lines = 1
	}
	_, shaped := ui.ShapeText(display, c.cardFont(), c.textWidthForSlip(width)).Size()
	return lines, c.windowHeight(c.textBlockPx(lines, shaped))
}

// textBlockPx is the text view's height for this many lines: the line box
// (ascent, descent, leading) times the line count, or the shaped height when
// a fallback face is taller, plus the text container's vertical inset.
func (c Card) textBlockPx(lines int, shaped float32) int {
	if lines < 1 {
		lines = 1
	}
	used := lines * c.LineBoxPx()
	if shapedPx := int(math.Ceil(float64(shaped))); shapedPx > used {
		used = shapedPx
	}
	used += int(math.Ceil(2 * TextContainerInsetY))
	if used < 1 {
		return 1
	}
	return used
}

// windowHeight adds the scroll view's vertical inset on both edges.
func (c Card) windowHeight(textBlock int) int {
	h := textBlock + 2*c.FrameInsetY()
	if h < 1 {
		return 1
	}
	return h
}

// OneLineHeight is the shortest window that still shows one line at width,
// including descent, leading, and the text view insets.
func (c Card) OneLineHeight(int) int {
	return c.windowHeight(c.textBlockPx(1, c.lineBox()))
}

func (c Card) clampWidth(display string, width int) int {
	maxW := c.EffectiveMaxWidth()
	if width <= 0 {
		width = c.naturalSlipWidth(display)
	}
	if width > maxW {
		width = maxW
	}
	if width < 1 {
		width = 1
	}
	return width
}

func (c Card) maxWindowHeight(one int) int {
	maxH := c.EffectiveMaxHeight()
	if maxH < one {
		return one
	}
	return maxH
}

// Layout sizes a new card around the whole string. Nothing is ellipsized.
// The window is the measured height, capped at max height; past that the
// card scrolls.
func (c Card) Layout(display string, width, _ int) CardLayout {
	return c.layout(display, width, 0, true)
}

// LayoutViewport fits display into an explicit window height. A viewport
// shorter than the text scrolls. A viewport taller than the text shrinks
// to the text. Height stays inside one line and max height.
func (c Card) LayoutViewport(display string, width, viewport int) CardLayout {
	return c.layout(display, width, viewport, false)
}

func (c Card) layout(display string, width, viewport int, natural bool) CardLayout {
	if !utf8.ValidString(display) {
		display = string([]rune(display))
	}
	width = c.clampWidth(display, width)
	lines, contentH := c.measureBox(display, width)
	one := c.OneLineHeight(width)
	if contentH < one {
		contentH = one
	}
	maxH := c.maxWindowHeight(one)
	height := contentH
	if !natural {
		if viewport < one {
			viewport = one
		}
		height = viewport
		if height > contentH {
			height = contentH
		}
	}
	if height > maxH {
		height = maxH
	}
	if height < one {
		height = one
	}
	return CardLayout{
		Text:         display,
		Width:        width,
		Height:       height,
		VisibleLines: lines,
		FullLines:    lines,
		Truncated:    false,
		Scrolls:      contentH > height,
	}
}

func (c Card) shapedHeight(display string, slipWidth int) (lines int, heightPx int) {
	layout := c.Layout(display, slipWidth, 0)
	return layout.VisibleLines, layout.Height
}

func (c Card) naturalSlipWidth(display string) int {
	// Width 0 is ui.Text's max-content measure: break only at newlines and
	// use the ink width. A huge wrap width is not the same number, and
	// min_width is a resize floor, not part of this measurement.
	w, _ := ui.ShapeText(display, c.cardFont(), 0).Size()
	chrome := 2*float64(c.FrameInsetX()) + 2*TextContainerInsetX + 2*LineFragmentPadding
	est := int(math.Ceil(float64(w) + chrome))
	if est < 1 {
		return 1
	}
	return est
}
