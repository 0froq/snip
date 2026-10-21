package macplace

import (
	"fmt"
	"math"

	"github.com/0froq/snip/config"
)

// windowEdge is one side of a window in AppKit's bottom-left coordinates.
type windowEdge int

const (
	windowEdgeLeft windowEdge = iota
	windowEdgeRight
	windowEdgeBottom
	windowEdgeTop
)

// dragWindowEdge is the frame after growing one edge by delta points.
// Left and bottom move the origin. Top and right change only the size.
func dragWindowEdge(x, y, w, h, delta float64, edge windowEdge) (float64, float64, float64, float64) {
	switch edge {
	case windowEdgeLeft:
		return x - delta, y, w + delta, h
	case windowEdgeRight:
		return x, y, w + delta, h
	case windowEdgeBottom:
		return x, y - delta, w, h + delta
	case windowEdgeTop:
		return x, y, w, h + delta
	default:
		panic(fmt.Sprintf("snip: unhandled window edge %d", int(edge)))
	}
}

// contentSizeChanged reports that a content view's width or height changed.
// A non-positive read is not a size. A move that keeps the size is not a
// size change; contentFrameChanged is what also notices the origin.
func contentSizeChanged(oldW, oldH, newW, newH float64) bool {
	if newW <= 0 || newH <= 0 {
		return false
	}
	return newW != oldW || newH != oldH
}

// contentFrameChanged reports that the content view moved or resized inside
// the window. A non-positive size is a failed read. A window move that leaves
// this frame alone does not report a change. A borderless content view stays
// at (0, 0), so a left or bottom drag is not one of these changes; that drag
// is liveStepNeedsRelayout on the window frame.
func contentFrameChanged(oldX, oldY, oldW, oldH, newX, newY, newW, newH float64) bool {
	if newW <= 0 || newH <= 0 {
		return false
	}
	return newX != oldX || newY != oldY || newW != oldW || newH != oldH
}

// liveStepNeedsRelayout reports that this live resize step has to place the
// text again. A borderless content view's origin stays (0, 0) in window
// coordinates, so a left or bottom drag does not change that view's frame
// even though the window origin moved. The window frame is the signal:
// its size changed, or the user is live-resizing and its origin moved.
// A pure move (not live, same size) is not a relayout.
func liveStepNeedsRelayout(live bool, oldX, oldY, oldW, oldH, newX, newY, newW, newH float64) bool {
	if newW <= 0 || newH <= 0 {
		return false
	}
	if newW != oldW || newH != oldH {
		return true
	}
	if !live {
		return false
	}
	return newX != oldX || newY != oldY
}

// windowDrag is one edge or corner, in AppKit's bottom-left coordinates.
type windowDrag int

const (
	windowDragTop windowDrag = iota
	windowDragBottom
	windowDragLeft
	windowDragRight
	windowDragTopLeft
	windowDragTopRight
	windowDragBottomLeft
	windowDragBottomRight
)

// dragWindow is the frame after dragging one edge or corner by dx and dy.
// Left and bottom move the origin. Top and right change only the size.
// A corner does both of its edges.
func dragWindow(x, y, w, h, dx, dy float64, dir windowDrag) (float64, float64, float64, float64) {
	switch dir {
	case windowDragTop:
		return dragWindowEdge(x, y, w, h, dy, windowEdgeTop)
	case windowDragBottom:
		return dragWindowEdge(x, y, w, h, dy, windowEdgeBottom)
	case windowDragLeft:
		return dragWindowEdge(x, y, w, h, dx, windowEdgeLeft)
	case windowDragRight:
		return dragWindowEdge(x, y, w, h, dx, windowEdgeRight)
	case windowDragTopLeft:
		x, y, w, h = dragWindowEdge(x, y, w, h, dx, windowEdgeLeft)
		return dragWindowEdge(x, y, w, h, dy, windowEdgeTop)
	case windowDragTopRight:
		x, y, w, h = dragWindowEdge(x, y, w, h, dx, windowEdgeRight)
		return dragWindowEdge(x, y, w, h, dy, windowEdgeTop)
	case windowDragBottomLeft:
		x, y, w, h = dragWindowEdge(x, y, w, h, dx, windowEdgeLeft)
		return dragWindowEdge(x, y, w, h, dy, windowEdgeBottom)
	case windowDragBottomRight:
		x, y, w, h = dragWindowEdge(x, y, w, h, dx, windowEdgeRight)
		return dragWindowEdge(x, y, w, h, dy, windowEdgeBottom)
	default:
		panic(fmt.Sprintf("snip: unhandled window drag %d", int(dir)))
	}
}

// draggedCardLayout is where the content view, the text, and the scroll
// offset sit after one drag. The content view is pinned to the window, so a
// bottom or left drag moves the window origin and does not move the view
// inside it. Text is in the content view's unflipped coordinates: its top is
// the window height minus the top inset. ScrollX/ScrollY are the clip origin.
// The text view is flipped, so Y is the distance from the top of the document.
type draggedCardLayout struct {
	WinX, WinY, WinW, WinH                 float64
	ContentX, ContentY, ContentW, ContentH float64
	TextX, TextY, TextW, TextH             float64
	PadLeft, PadTop, PadRight, PadBottom   int
	ContainerW                             float64
	ScrollX, ScrollY                       float64
	OK                                     bool
}

// layoutDraggedCard places the card after dragging dir by dx and dy.
// clipY is the flipped scroll offset from before the drag. A change in the
// window origin is not added to it.
func layoutDraggedCard(x, y, w, h, dx, dy float64, dir windowDrag, padX, padY, border, resizeMargin, topExtra int, docH, clipY float64) draggedCardLayout {
	nx, ny, nw, nh := dragWindow(x, y, w, h, dx, dy, dir)
	lay := layoutCardFrame(nx, ny, nw, nh, padX, padY, border, resizeMargin, topExtra, docH, clipY, true, clipY-(ny-y), nx-x, ny-y)
	return lay
}

// layoutCardFrame pins the content view to (0,0,winW,winH) and places the
// text from the top padding. reported is the clip origin after the frame
// moved; originDX/originDY are the window-origin change. haveClip is false
// on the first layout, which trusts reported.
func layoutCardFrame(winX, winY, winW, winH float64, padX, padY, border, resizeMargin, topExtra int, docH, clipY float64, haveClip bool, reported, originDX, originDY float64) draggedCardLayout {
	lay := draggedCardLayout{WinX: winX, WinY: winY, WinW: winW, WinH: winH}
	lay.ContentX, lay.ContentY = 0, 0
	lay.ContentW, lay.ContentH = winW, winH
	left, top, right, bottom := config.TextFrameInsets(padX, padY, border, resizeMargin, topExtra)
	lay.PadLeft, lay.PadTop, lay.PadRight, lay.PadBottom = left, top, right, bottom
	tx, ty, tw, th, ok := TextContentFrame(int(math.Round(winW)), int(math.Round(winH)), padX, padY, border, resizeMargin, topExtra)
	if !ok {
		return lay
	}
	lay.OK = true
	lay.TextX, lay.TextY, lay.TextW, lay.TextH = tx, ty, tw, th
	inner := tw - 2*config.TextContainerInsetX
	if inner < 1 {
		inner = 1
	}
	lay.ContainerW = inner
	lay.ScrollX = 0
	lay.ScrollY = adjustedClipOrigin(clipY, haveClip, reported, originDX, originDY, docH, th)
	return lay
}

// box is an axis-aligned rectangle in the same coordinate space as the frame
// it is compared with. AppKit frames are bottom-left; mygo frames are top-left.
// The slide only cares that both use one space.
type box struct {
	X, Y, W, H float64
}

// snapResizeFrame replaces a window's size and keeps its top-left corner.
// AppKit's origin is the bottom-left, so the top edge is originY+height and
// only origin.y moves when the height changes. origin.x stays. A zero,
// negative, or non-finite source frame is not a place to rebuild from: ok
// is false and the caller must not setFrame. A zero visible rect is not a
// destination either; the anchored origin is left alone.
func snapResizeFrame(originX, originY, width, height float64, newW, newH, minW, minH int, visible box) (x, y, w, h float64, ok bool) {
	if !usableFrame(originX, originY, width, height) {
		return originX, originY, width, height, false
	}
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
	// Top-left anchor: left edge stays, top edge stays.
	x = originX
	y = originY + height - h
	if !finite(x) || !finite(y) {
		return originX, originY, width, height, false
	}
	if !usableFrame(visible.X, visible.Y, visible.W, visible.H) {
		return x, y, w, h, true
	}
	x, y, w, h = slideToIntersect(x, y, w, h, visible)
	if !finite(x) || !finite(y) {
		return originX, originY, width, height, false
	}
	return x, y, w, h, true
}

func usableFrame(x, y, w, h float64) bool {
	return finite(x) && finite(y) && finite(w) && finite(h) && w > 0 && h > 0
}

func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
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
