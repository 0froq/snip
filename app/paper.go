package app

import "github.com/egoist/mygo/ui"

func paperInk(dark bool) ui.Color {
	if dark {
		return ui.Hex("#E7DDD0")
	}
	return ui.Hex("#2A241C")
}

func paperEdge(dark bool) ui.Color {
	// 60% of the previous edge and 40% of the card surface.
	if dark {
		return ui.Hex("#4B4038")
	}
	return ui.Hex("#DFD3C3")
}

func paperStroke(dark bool) ui.Color {
	return paperEdge(dark).Alpha(0.25)
}

func paperShadow(dark bool) ui.Color {
	if dark {
		return ui.Hex("#000000").Alpha(0.42)
	}
	return ui.Hex("#3A2E24").Alpha(0.16)
}
