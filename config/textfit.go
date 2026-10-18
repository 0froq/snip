package config

import (
	"math"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

const cardBorderPx = 1

// CardBorderPx is the slip border width in the card view.
func CardBorderPx() int { return cardBorderPx }

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
	inner := slipWidth - 2*c.EffectivePaddingX() - 2*cardBorderPx
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
	_, textH := ui.ShapeText(display, c.cardFont(), c.textWidthForSlip(width)).Size()
	height = int(math.Ceil(float64(textH))) + 2*c.EffectivePaddingY() + 2*cardBorderPx
	if height < 1 {
		height = 1
	}
	return lines, height
}

// OneLineHeight is the shortest window that still shows one line at width.
func (c Card) OneLineHeight(width int) int {
	h := int(math.Ceil(float64(c.lineBox()))) + 2*c.EffectivePaddingY() + 2*cardBorderPx
	if h < 1 {
		return 1
	}
	return h
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
	est := int(math.Ceil(float64(w))) + 2*c.EffectivePaddingX() + 2*cardBorderPx
	if est < 1 {
		return 1
	}
	return est
}
