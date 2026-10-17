package macplace

import "math"

// box is an axis-aligned rectangle in the same coordinate space as the frame
// it is compared with. AppKit frames are bottom-left; mygo frames are top-left.
// The slide only cares that both use one space.
type box struct {
	X, Y, W, H float64
}

// snapResizeFrame replaces a window's size without letting it vanish.
// The top and left edges stay put (AppKit's bottom-left origin means the top
// is originY+height), then the frame is slid the minimum amount so it
// intersects visible. A fast shrink below minW/minH is clamped up. A zero
// visible rect skips the slide, so callers without a screen still get a
// positive size.
func snapResizeFrame(originX, originY, _, height float64, newW, newH, minW, minH int, visible box) (x, y, w, h float64) {
	if minW < 1 {
		minW = 1
	}
	if minH < 1 {
		minH = 1
	}
	if newW < minW {
		newW = minW
	}
	if newH < minH {
		newH = minH
	}
	w = float64(newW)
	h = float64(newH)
	top := originY + height
	x = originX
	y = top - h
	if visible.W <= 0 || visible.H <= 0 {
		return x, y, w, h
	}
	return slideToIntersect(x, y, w, h, visible)
}

// slideToIntersect moves a frame onto visible only when the two do not meet.
// A frame that already crosses the rect, including one that hangs off an edge,
// is left where it is so an in-progress drag is not fought.
func slideToIntersect(x, y, w, h float64, visible box) (float64, float64, float64, float64) {
	if visible.W <= 0 || visible.H <= 0 || w <= 0 || h <= 0 {
		return x, y, w, h
	}
	if intersects(x, y, w, h, visible) {
		return x, y, w, h
	}
	vx, vy := visible.X, visible.Y
	vw, vh := visible.W, visible.H
	if x+w <= vx {
		x = vx
	} else if x >= vx+vw {
		span := w
		if span > vw {
			span = vw
		}
		x = vx + vw - span
	}
	if y+h <= vy {
		y = vy
	} else if y >= vy+vh {
		span := h
		if span > vh {
			span = vh
		}
		y = vy + vh - span
	}
	return x, y, w, h
}

func intersects(x, y, w, h float64, b box) bool {
	return w > 0 && h > 0 && b.W > 0 && b.H > 0 &&
		x < b.X+b.W && x+w > b.X && y < b.Y+b.H && y+h > b.Y
}

// nearestVisible returns the first frame the rect already meets, otherwise
// the frame whose center is closest. An empty list returns a zero box, which
// tells snapResizeFrame to skip the screen slide.
func nearestVisible(frames []box, x, y, w, h float64) box {
	if len(frames) == 0 {
		return box{}
	}
	best := frames[0]
	bestDist := math.MaxFloat64
	have := false
	for _, f := range frames {
		if f.W <= 0 || f.H <= 0 {
			continue
		}
		if intersects(x, y, w, h, f) {
			return f
		}
		d := centerDist(x, y, w, h, f)
		if !have || d < bestDist {
			best, bestDist, have = f, d, true
		}
	}
	if !have {
		return box{}
	}
	return best
}

func centerDist(x, y, w, h float64, b box) float64 {
	cx, cy := x+w/2, y+h/2
	bx, by := b.X+b.W/2, b.Y+b.H/2
	dx, dy := cx-bx, cy-by
	return dx*dx + dy*dy
}

// SlideIntoVisible moves an integer frame onto the nearest area when it does
// not already intersect one. Areas use the same origin as x,y. An empty list,
// or areas with no positive size, leaves the origin unchanged.
func SlideIntoVisible(x, y, w, h int, areas []Rect) (int, int) {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	frames := make([]box, 0, len(areas))
	for _, a := range areas {
		if a.Width <= 0 || a.Height <= 0 {
			continue
		}
		frames = append(frames, box{
			X: float64(a.X), Y: float64(a.Y),
			W: float64(a.Width), H: float64(a.Height),
		})
	}
	visible := nearestVisible(frames, float64(x), float64(y), float64(w), float64(h))
	nx, ny, _, _ := slideToIntersect(float64(x), float64(y), float64(w), float64(h), visible)
	return int(math.Round(nx)), int(math.Round(ny))
}
