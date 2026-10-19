package config

import (
	"math"
	"sort"
	"strings"
	"unicode/utf8"

	"git.zacharyzhang.com/ZacharyZhang-NY/8bitMyGoUI/pixel"
	"github.com/egoist/mygo/ui"
)

// cardBorderPx matches a 3-dot pixel window frame at 2 DIPs per dot.
const cardBorderPx = 6

// CardTitleStripPx is the top band kept clear for the close control.
// Twelve dots at 2 DIPs: the frame, a gap, a 7-dot button, and a gap.
const CardTitleStripPx = 24

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
	// lineFitSlack keeps the fragment strictly wider than the ink. A line
	// that fills the fragment exactly wraps the last word when the text
	// view's advances are a fraction over the measure.
	lineFitSlack = 1.0
)

// CardBorderPx is the slip border width in the card view.
func CardBorderPx() int { return cardBorderPx }

// FrameInsetX is how far the text view sits from the left and right edges.
func (c Card) FrameInsetX() int {
	left, _, _, _ := c.edgeInsets()
	return left
}

// FrameInsetY is how far the text view sits from the bottom edge.
func (c Card) FrameInsetY() int {
	_, _, _, bottom := c.edgeInsets()
	return bottom
}

// TextFrameInsets is the scroll view's distance from each window edge.
// Left, right, and bottom are the pixel frame plus padding, and at least
// the resize band. Top is that same band, or the title strip when the
// close control is taller, so the line starts below the button.
func TextFrameInsets(padX, padY, border, resizeMargin, topExtra int) (left, top, right, bottom int) {
	left = padX + border
	bottom = padY + border
	if resizeMargin > left {
		left = resizeMargin
	}
	if resizeMargin > bottom {
		bottom = resizeMargin
	}
	right = left
	top = bottom
	if topExtra > top {
		top = topExtra
	}
	return left, top, right, bottom
}

func (c Card) edgeInsets() (left, top, right, bottom int) {
	return TextFrameInsets(c.EffectivePaddingX(), c.EffectivePaddingY(), cardBorderPx, CardResizeMargin, CardTitleStripPx)
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
	// The text view draws ResolveFont's face. Measuring the whole CSS list
	// would size a leading concrete family with a later, narrower fallback.
	font := c.UIFont(c.EffectiveFontSize())
	if name, system := ResolveFont(font.Family); !system && name != "" {
		font.Family = name
	}
	return font
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
	left, _, right, _ := c.edgeInsets()
	inner := float64(slipWidth-left-right) - 2*TextContainerInsetX - 2*LineFragmentPadding
	if inner < 1 {
		return 1
	}
	return float32(inner)
}

func (c Card) lineBox() float32 {
	if c.PixelFont() {
		return float32(c.pixelLinePx())
	}
	m := c.cardFont().Metrics()
	h := m.Ascent + m.Descent + m.LineGap
	if h < 1 {
		return float32(c.EffectiveLineHeight())
	}
	return h
}

// pixelLinePx is one bitmap line in DIPs at the card's scale.
func (c Card) pixelLinePx() int {
	return pixel.LineHeight * c.pixelUnit()
}

// pixelUnit is DIPs per font dot. Scale 2 doubles the face and the measure.
func (c Card) pixelUnit() int {
	unit := c.PixelScale() * PixelDot
	if unit < 1 {
		return 1
	}
	return unit
}

// pixelWrapDots is the width the bitmap wraps at, in font dots.
func (c Card) pixelWrapDots(slipWidth int) int {
	left, _, right, _ := c.edgeInsets()
	inner := slipWidth - left - right
	if inner < 1 {
		return 1
	}
	dots := inner / c.pixelUnit()
	if dots < 1 {
		return 1
	}
	return dots
}

func (c Card) pixelLineCount(text string, width int) int {
	if text == "" {
		return 1
	}
	n := len(pixel.Lines(text, c.pixelWrapDots(width)))
	if n < 1 {
		return 1
	}
	return n
}

func (c Card) pixelSlipWidth(display string) int {
	maxW := 0
	for _, line := range pixel.Lines(display, 0) {
		if w := pixel.Measure(line); w > maxW {
			maxW = w
		}
	}
	left, _, right, _ := c.edgeInsets()
	est := maxW*c.pixelUnit() + left + right
	if est < 1 {
		return 1
	}
	return est
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
	if c.PixelFont() {
		return c.pixelLineCount(text, width)
	}
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
	if c.PixelFont() {
		lines = c.pixelLineCount(display, width)
		return lines, c.windowHeight(c.textBlockPx(lines, float32(lines*c.pixelLinePx())))
	}
	lines = c.linesForText(display, width)
	if lines < 1 {
		lines = 1
	}
	_, shaped := ui.ShapeText(display, c.cardFont(), c.textWidthForSlip(width)).Size()
	return lines, c.windowHeight(c.textBlockPx(lines, shaped))
}

// textBlockPx is the NSTextView document height for this many lines: the
// line box (ascent, descent, leading) times the line count, or the shaped
// height when a fallback face is taller, plus the text container's vertical
// inset. One extra point covers the used rect running a fraction past that
// box; without it the scroll view shows a scrollbar on a single line.
func (c Card) textBlockPx(lines int, shaped float32) int {
	if lines < 1 {
		lines = 1
	}
	used := lines * c.LineBoxPx()
	if shapedPx := int(math.Ceil(float64(shaped))); shapedPx > used {
		used = shapedPx
	}
	used += int(math.Ceil(2*TextContainerInsetY + lineFitSlack))
	if used < 1 {
		return 1
	}
	return used
}

// TopInset is the text's top inset. The close strip wins when it is taller
// than the padding, so a line never sits under the button.
func (c Card) TopInset() int {
	_, top, _, _ := c.edgeInsets()
	return top
}

// windowHeight adds the top strip and the bottom inset around the text.
func (c Card) windowHeight(textBlock int) int {
	h := textBlock + c.TopInset() + c.FrameInsetY()
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
	if c.PixelFont() {
		return c.pixelSlipWidth(display)
	}
	// Width and height share cardFont and edgeInsets. Width 0 is ui.Text's
	// max-content measure: break only at newlines and use the ink width.
	// A huge wrap width is not the same number, and min_width is a resize
	// floor, not part of this measurement.
	w, _ := ui.ShapeText(display, c.cardFont(), 0).Size()
	left, _, right, _ := c.edgeInsets()
	chrome := float64(left+right) + 2*TextContainerInsetX + 2*LineFragmentPadding
	est := int(math.Ceil(float64(w) + chrome + lineFitSlack))
	if est < 1 {
		return 1
	}
	return est
}
