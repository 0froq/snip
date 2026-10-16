package app

import "github.com/egoist/mygo/ui"

func paperTheme(dark bool, active ui.Color) *ui.Theme {
	t := ui.LightTheme()
	if dark {
		t = ui.DarkTheme()
		t.Background = ui.Hex("#241F1B")
		t.Surface = ui.Hex("#3A322C")
		t.SurfaceHover = ui.Hex("#463E36")
		t.Border = ui.Hex("#5C5148")
		t.Text = ui.Hex("#F6F0E8")
		t.TextMuted = ui.Hex("#C4B6A6")
	} else {
		t.Background = ui.Hex("#F3EBDF")
		t.Surface = ui.Hex("#FFF9F1")
		t.SurfaceHover = ui.Hex("#F7F0E4")
		t.Border = ui.Hex("#E4D5C3")
		t.Text = ui.Hex("#3B3128")
		t.TextMuted = ui.Hex("#8A7768")
	}
	t.Accent = active
	t.Radius = 12
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
