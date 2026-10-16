package app

import "github.com/egoist/mygo/ui"

func paperTheme(dark bool, active ui.Color) *ui.Theme {
	t := ui.LightTheme()
	if dark {
		t = ui.DarkTheme()
		t.Background = ui.Hex("#1A1613")
		t.Surface = ui.Hex("#312924")
		t.SurfaceHover = ui.Hex("#3C332C")
		t.Border = ui.Hex("#5C5046")
		t.Text = paperInk(true)
		t.TextMuted = ui.Hex("#B6A694")
	} else {
		t.Background = ui.Hex("#E6D9C8")
		t.Surface = ui.Hex("#F3EADF")
		t.SurfaceHover = ui.Hex("#E9DFD0")
		t.Border = ui.Hex("#D2C3B0")
		t.Text = paperInk(false)
		t.TextMuted = ui.Hex("#8A7764")
	}
	t.Accent = active
	t.Radius = 3
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
