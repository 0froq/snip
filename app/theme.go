package app

import (
	"github.com/0froq/snip/internal/macplace"
	"github.com/egoist/mygo/ui"
)

func paperTheme(dark bool, active ui.Color, font ui.Font) *ui.Theme {
	t := ui.LightTheme()
	if dark {
		t = ui.DarkTheme()
		t.Background = ui.Hex("#1A1613")
		t.Surface = ui.Hex("#312924")
		t.SurfaceHover = ui.Hex("#3C332C")
		t.Text = paperInk(true)
		t.TextMuted = ui.Hex("#B6A694")
	} else {
		t.Background = ui.Hex("#E6D9C8")
		t.Surface = ui.Hex("#F3EADF")
		t.SurfaceHover = ui.Hex("#E9DFD0")
		t.Text = paperInk(false)
		t.TextMuted = ui.Hex("#8A7764")
	}
	t.Border = paperEdge(dark)
	t.Accent = active
	t.Radius = float32(macplace.WindowCornerRadius)
	t.Font = font.Family
	if font.Size > 0 {
		t.FontSize = font.Size
	}
	return t
}

func activeColor(hex string, dark bool) ui.Color {
	if hex == "" {
		if dark {
			return ui.Hex("#E0A36A")
		}
		return ui.Hex("#C4784A")
	}
	return ui.Hex(hex)
}
