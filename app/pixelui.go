package app

import (
	"git.zacharyzhang.com/ZacharyZhang-NY/8bitMyGoUI/pixel"
	"github.com/egoist/mygo/ui"
)

// pixelDot is device-independent pixels per pixel-art dot.
const pixelDot = 2

// cardFrameDots is the thickness of a pixel.Window frame.
const cardFrameDots = 3

// usePixel installs the pixel kit for this frame. Call it before any
// pixel component. dark is the system appearance read before Use, because
// Use replaces the theme.
func usePixel(c *ui.Context, dark bool, lang string) pixel.Kit {
	pixel.Use(c, pixel.Settings{
		Palette: appearancePalette(dark),
		Dot:     pixelDot,
		Locale:  pixelLocale(lang),
	})
	return pixel.K(c)
}

// appearancePalette follows the system appearance until a saved palette
// overrides it. Autumn is the warm pair closest to the old paper.
func appearancePalette(dark bool) *pixel.Palette {
	if dark {
		return pixel.AutumnNight
	}
	return pixel.AutumnDay
}

func pixelLocale(lang string) pixel.Locale {
	switch lang {
	case "zh", "zh-CN", "zh-Hans":
		return pixel.Zh
	default:
		return pixel.En
	}
}

// drawCardChrome paints a stepped window frame. An active card's outer
// edge is Accent. An editor sinks a well inside the frame.
func drawCardChrome(g pixel.Canvas, active, editing bool) {
	if g.W < 3 || g.H < 3 {
		return
	}
	g.Frame(0, 0, g.W, g.H, pixel.Window, pixel.Face)
	edge := pixel.Edge
	if active {
		edge = pixel.Accent
	}
	steppedOutline(g, 0, 0, g.W, g.H, edge)
	if !editing || g.W < 9 || g.H < 9 {
		return
	}
	g.Frame(cardFrameDots, cardFrameDots, g.W-2*cardFrameDots, g.H-2*cardFrameDots, pixel.Sunken, pixel.Well)
}

// steppedOutline is a 1-dot notch, the same corner a pixel window leaves empty.
func steppedOutline(g pixel.Canvas, x, y, w, h int, s pixel.Slot) {
	if w < 3 || h < 3 {
		return
	}
	g.HLine(x+1, y, w-2, s)
	g.HLine(x+1, y+h-1, w-2, s)
	g.VLine(x, y+1, h-2, s)
	g.VLine(x+w-1, y+1, h-2, s)
}
