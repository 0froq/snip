package app

import "github.com/egoist/mygo/ui"

func paperSeed(id string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(id); i++ {
		h ^= uint32(id[i])
		h *= 16777619
	}
	return h
}

// paperRadius is the same on every corner so the slip matches a rectangular window.
const paperRadius = 10

func paperCorners(id string) (float32, float32, float32, float32) {
	_ = id
	return paperRadius, paperRadius, paperRadius, paperRadius
}

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

func paintPaperGrain(p *ui.Painter, r ui.Rect, dark bool, seed uint32) {
	if r.W < 4 || r.H < 4 {
		return
	}
	fiber := ui.Hex("#8D7760").Alpha(0.07)
	speck := ui.Hex("#FFF8EE").Alpha(0.05)
	if dark {
		fiber = ui.Hex("#140F0C").Alpha(0.22)
		speck = ui.Hex("#A89884").Alpha(0.05)
	}
	p.Clip(r, 4, func() {
		n := 8 + int(seed%5)
		for i := 0; i < n; i++ {
			h := seed*uint32(i+1)*2246822519 + 3266489917
			x := r.X + float32(h%uint32(r.W))
			y := r.Y + float32((h/17)%uint32(r.H))
			length := 5 + float32(h%8)
			dy := float32(int(h%3) - 1)
			p.Line(x, y, x+length, y+dy, 0.6, fiber)
			if i%3 == 0 {
				p.Line(x+1, y+2, x+2, y+2, 0.7, speck)
			}
		}
		edge := paperEdge(dark).Alpha(0.55)
		bite := 3 + int(seed%4)
		for i := 0; i < bite; i++ {
			h := seed*uint32(i+9) + 97
			x := r.X + 6 + float32(h%uint32(max(r.W-12, 1)))
			p.Line(x, r.Y+r.H-1, x+float32(2+h%3), r.Y+r.H-1, 0.8, edge)
		}
	})
}
