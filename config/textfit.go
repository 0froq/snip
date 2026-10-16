package config

import (
	"math"

	"github.com/egoist/mygo/ui"
)

const cardBorderPx = 1

// CardBorderPx is the slip border width in the card view.
func CardBorderPx() int { return cardBorderPx }

func (c Card) cardFont() ui.Font {
	return ui.Font{Family: "Georgia", Size: float32(c.EffectiveFontSize())}
}

func (c Card) textWidthForSlip(slipWidth int) float32 {
	inner := slipWidth - 2*c.EffectivePaddingX() - 2*cardBorderPx
	if inner < 1 {
		return 1
	}
	return float32(inner)
}

// LineBox is the Georgia line height the card text actually draws, in DIPs.
func (c Card) LineBox() float32 {
	m := c.cardFont().Metrics()
	h := m.Ascent + m.Descent + m.LineGap
	if h < 1 {
		return float32(c.EffectiveLineHeight())
	}
	return h
}

// WrappedLines is how many lines display needs at slipWidth, with no max_lines cap.
func (c Card) WrappedLines(display string, slipWidth int) int {
	textW := c.textWidthForSlip(slipWidth)
	layout := ui.ShapeText(display, c.cardFont(), textW)
	_, textH := layout.Size()
	line := float64(c.LineBox())
	if line < 1 {
		line = 1
	}
	lines := int(math.Round(float64(textH) / line))
	if lines < 1 {
		return 1
	}
	return lines
}

// LinesThatFit is the most whole lines that fit in max height.
func (c Card) LinesThatFit() int {
	line := float64(c.LineBox())
	if line < 1 {
		line = 1
	}
	inner := float64(c.EffectiveMaxHeight() - 2*c.EffectivePaddingY() - 2*cardBorderPx)
	n := int(inner / line)
	if n < 1 {
		return 1
	}
	return n
}

// VisibleLines is the line count to draw. chosen is the user's count, or 0 to use max_lines.
// full is the wrapped length of the text. The result never exceeds the text or max height.
func (c Card) VisibleLines(chosen, full int) int {
	capN := c.EffectiveMaxLines()
	if chosen > 0 {
		capN = chosen
	}
	n := full
	if n > capN {
		n = capN
	}
	if fit := c.LinesThatFit(); n > fit {
		n = fit
	}
	if n < 1 {
		return 1
	}
	return n
}

// HeightForLines is the slip height for a whole number of lines.
func (c Card) HeightForLines(lines int) int {
	if lines < 1 {
		lines = 1
	}
	if fit := c.LinesThatFit(); lines > fit {
		lines = fit
	}
	line := float64(c.LineBox())
	return int(math.Ceil(float64(lines)*line)) + 2*c.EffectivePaddingY() + 2*cardBorderPx
}

func (c Card) shapedHeight(display string, slipWidth int) (lines int, heightPx int) {
	full := c.WrappedLines(display, slipWidth)
	lines = c.VisibleLines(0, full)
	return lines, c.HeightForLines(lines)
}

func (c Card) naturalSlipWidth(display string) int {
	layout := ui.ShapeText(display, c.cardFont(), 1e6)
	w, _ := layout.Size()
	est := int(math.Ceil(float64(w))) + 2*c.EffectivePaddingX() + 2*cardBorderPx
	return est
}
