package app

import (
	"git.zacharyzhang.com/ZacharyZhang-NY/8bitMyGoUI/pixel"
	"github.com/0froq/snip/config"
	"github.com/egoist/mygo/ui"
)

// pixelDot is device-independent pixels per pixel-art dot.
const pixelDot = 2

// cardFrameDots is the thickness of a pixel.Window frame.
const cardFrameDots = 3

// usePixel installs the pixel kit for this frame. Call it before any
// pixel component. dark is the system appearance read before Use, because
// Use replaces the theme. Auto maps that appearance to Autumn Day or Autumn Night.
func usePixel(c *ui.Context, cfg config.Config, dark bool) pixel.Kit {
	pixel.Use(c, pixel.Settings{
		Palette: resolvePalette(cfg.Theme.EffectivePalette(), dark),
		Dot:     pixelDot,
		Locale:  pixelLocale(cfg.Language),
	})
	return pixel.K(c)
}

func resolvePalette(name string, dark bool) *pixel.Palette {
	switch name {
	case config.PaletteAuto, "":
		return appearancePalette(dark)
	case config.PaletteSpringDay:
		return pixel.SpringDay
	case config.PaletteSpringNight:
		return pixel.SpringNight
	case config.PaletteSummerDay:
		return pixel.SummerDay
	case config.PaletteSummerNight:
		return pixel.SummerNight
	case config.PaletteAutumnDay:
		return pixel.AutumnDay
	case config.PaletteAutumnNight:
		return pixel.AutumnNight
	case config.PaletteWinterDay:
		return pixel.WinterDay
	case config.PaletteWinterNight:
		return pixel.WinterNight
	default:
		return appearancePalette(dark)
	}
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
