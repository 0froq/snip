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

func (c Card) shapedHeight(display string, slipWidth int) (lines int, heightPx int) {
	textW := c.textWidthForSlip(slipWidth)
	layout := ui.ShapeText(display, c.cardFont(), textW)
	_, textH := layout.Size()
	lineH := float32(c.EffectiveLineHeight())
	if lineH < 1 {
		lineH = 1
	}
	lines = int(math.Ceil(float64(textH) / float64(lineH)))
	if lines < 1 {
		lines = 1
	}
	maxLines := c.EffectiveMaxLines()
	if lines > maxLines {
		lines = maxLines
	}
	heightPx = int(math.Ceil(float64(lines)*float64(lineH))) + 2*c.EffectivePaddingY() + 2*cardBorderPx
	return lines, heightPx
}

func (c Card) naturalSlipWidth(display string) int {
	layout := ui.ShapeText(display, c.cardFont(), 1e6)
	w, _ := layout.Size()
	est := int(math.Ceil(float64(w))) + 2*c.EffectivePaddingX() + 2*cardBorderPx
	return est
}
