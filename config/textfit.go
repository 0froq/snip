package config

import (
	"math"
	"unicode/utf8"

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
}

func (c Card) linesForText(text string, width int) int {
	_, h := ui.ShapeText(text, c.cardFont(), c.textWidthForSlip(width)).Size()
	line := float64(c.lineBox())
	if line < 1 {
		line = 1
	}
	n := int(math.Round(float64(h) / line))
	if n < 1 {
		return 1
	}
	return n
}

func (c Card) maxFittingLines() int {
	line := float64(c.lineBox())
	inside := float64(c.EffectiveMaxHeight() - 2*c.EffectivePaddingY() - 2*cardBorderPx)
	n := int(math.Floor(inside / line))
	if n < 1 {
		return 1
	}
	return n
}

func (c Card) truncate(display string, width, lines int) (string, bool) {
	if c.linesForText(display, width) <= lines {
		return display, false
	}
	runes := []rune(display)
	lo, hi := 0, len(runes)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		candidate := string(runes[:mid]) + "…"
		if c.linesForText(candidate, width) <= lines {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	if lo == 0 {
		return "…", true
	}
	return string(runes[:lo]) + "…", true
}

// Layout truncates first, then measures the exact text that the card draws.
// A zero width chooses a natural one-line width, capped at max width.
// A zero visibleLines uses the configured max line count.
func (c Card) Layout(display string, width, visibleLines int) CardLayout {
	if !utf8.ValidString(display) {
		display = string([]rune(display))
	}
	minW, maxW := c.EffectiveMinWidth(), c.EffectiveMaxWidth()
	naturalWidth := width <= 0
	if width <= 0 {
		width = c.naturalSlipWidth(display)
	}
	if width < minW {
		width = minW
	}
	if width > maxW {
		width = maxW
	}
	full := c.linesForText(display, width)
	if visibleLines <= 0 {
		visibleLines = c.EffectiveMaxLines()
	}
	if visibleLines > full {
		visibleLines = full
	}
	if fit := c.maxFittingLines(); visibleLines > fit {
		visibleLines = fit
	}
	if visibleLines < 1 {
		visibleLines = 1
	}
	text, truncated := c.truncate(display, width, visibleLines)
	textW, textH := ui.ShapeText(text, c.cardFont(), c.textWidthForSlip(width)).Size()
	if naturalWidth && c.naturalSlipWidth(display) <= maxW {
		measured := int(math.Ceil(float64(textW))) + 2*c.EffectivePaddingX() + 2*cardBorderPx
		if measured > minW && measured < width {
			width = measured
		}
	}
	height := int(math.Ceil(float64(textH))) + 2*c.EffectivePaddingY() + 2*cardBorderPx
	if height > c.EffectiveMaxHeight() {
		height = c.EffectiveMaxHeight()
	}
	return CardLayout{
		Text:         text,
		Width:        width,
		Height:       height,
		VisibleLines: visibleLines,
		FullLines:    full,
		Truncated:    truncated,
	}
}

func (c Card) shapedHeight(display string, slipWidth int) (lines int, heightPx int) {
	layout := c.Layout(display, slipWidth, 0)
	return layout.VisibleLines, layout.Height
}

func (c Card) naturalSlipWidth(display string) int {
	layout := ui.ShapeText(display, c.cardFont(), 1e6)
	w, _ := layout.Size()
	est := int(math.Ceil(float64(w))) + 2*c.EffectivePaddingX() + 2*cardBorderPx
	return est
}
