package app

import "github.com/egoist/mygo/ui"

func paperInk(dark bool) ui.Color {
	if dark {
		return ui.Hex("#E7DDD0")
	}
	return ui.Hex("#2A241C")
}

func paperEdge(dark bool) ui.Color {
	if dark {
		return ui.Hex("#5C5046")
	}
	return ui.Hex("#D2C3B0")
}

func paperShadow(dark bool) ui.Color {
	if dark {
		return ui.Hex("#000000").Alpha(0.42)
	}
	return ui.Hex("#3A2E24").Alpha(0.16)
}
