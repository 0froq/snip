package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const pointerSlop = 6.0

// Leave all four native resize borders alone, including the top corners.
// This is only a drag-to-move exclusion, not a resize grip or hit tracker.
const nativeResizeMargin = 10

const (
	edgeLeft = 1 << iota
	edgeRight
	edgeBottom
)

// cardEdge reports which resize edges a window-local point is on. The
// native AppKit border is only a few points wide, so the card handles its
// own wider band (nativeResizeMargin) on the left, right and bottom.
func cardEdge(x, y, width, height int) int {
	e := 0
	if x < nativeResizeMargin {
		e |= edgeLeft
	} else if x >= width-nativeResizeMargin {
		e |= edgeRight
	}
	if y >= height-nativeResizeMargin {
		e |= edgeBottom
	}
	return e
}

func edgeCursor(e int) ui.Cursor {
	switch {
	case e&edgeBottom != 0 && e&edgeRight != 0:
		return ui.CursorResizeNWSE
	case e&edgeBottom != 0 && e&edgeLeft != 0:
		return ui.CursorResizeNESW
	case e&edgeBottom != 0:
		return ui.CursorResizeNS
	case e != 0:
		return ui.CursorResizeEW
	}
	return ui.CursorDefault
}

func cardMoveInterior(x, y, width, height int) bool {
	return x >= nativeResizeMargin && y >= nativeResizeMargin &&
		x < width-nativeResizeMargin && y < height-nativeResizeMargin
}

// cardPointerInput moves the card window on drag and selects on a tap.
// HandleInput must consume the pointer stream because DragWindow skips
// press tracking, so Clicked and release handlers never run.
// Position follows mygo.Screen.CursorScreenPoint so moves stay 1:1 with the
// pointer (window-local coordinates shift as the window moves).
func cardPointerInput(a *App, host *cardUI, win *mygo.Window, col ui.Element) ui.Element {
	return col.HandleInput(func(ev ui.InputEvent) bool {
		switch ev.Kind {
		case ui.InputPointerDown:
			if ev.Button != 0 {
				return false
			}
			sp := mygo.Screen.CursorScreenPoint()
			wx, wy := win.Position()
			ww, wh := win.Size()
			if e := cardEdge(sp.X-wx, sp.Y-wy, ww, wh); e != 0 {
				host.resizeEdge = e
				host.pointerActive = true
				host.pressScreenX, host.pressScreenY = sp.X, sp.Y
				host.startX, host.startY, host.startW, host.startH = wx, wy, ww, wh
				return true
			}
			if !cardMoveInterior(sp.X-wx, sp.Y-wy, ww, wh) {
				return false
			}
			host.pointerActive = true
			host.pointerDragged = false
			host.pressClicks = ev.Clicks
			host.pressScreenX, host.pressScreenY = sp.X, sp.Y
			host.dragOffX = sp.X - wx
			host.dragOffY = sp.Y - wy
			return true
		case ui.InputPointerMove:
			sp := mygo.Screen.CursorScreenPoint()
			if !host.pointerActive {
				wx, wy := win.Position()
				ww, wh := win.Size()
				if e := cardEdge(sp.X-wx, sp.Y-wy, ww, wh); e != host.hoverEdge {
					host.hoverEdge = e
					win.Invalidate()
				}
				return false
			}
			dx := sp.X - host.pressScreenX
			dy := sp.Y - host.pressScreenY
			if e := host.resizeEdge; e != 0 {
				w, h := host.startW, host.startH
				if e&edgeRight != 0 {
					w += dx
				}
				if e&edgeLeft != 0 {
					w -= dx
				}
				if e&edgeBottom != 0 {
					h += dy
				}
				l := a.fitCardSize(host, w, h)
				win.SetSize(l.Width, l.Height)
				if e&edgeLeft != 0 {
					win.SetPosition(host.startX+host.startW-l.Width, host.startY)
				}
				win.Invalidate()
				return true
			}
			if !host.pointerDragged {
				if dx*dx+dy*dy < pointerSlop*pointerSlop {
					return true
				}
				host.pointerDragged = true
			}
			nx, ny := sp.X-host.dragOffX, sp.Y-host.dragOffY
			cx, cy := win.Position()
			if nx != cx || ny != cy {
				win.SetPosition(nx, ny)
			}
			return true
		case ui.InputPointerUp:
			if ev.Button != 0 || !host.pointerActive {
				return false
			}
			host.pointerActive = false
			if host.resizeEdge != 0 {
				host.resizeEdge = 0
				a.syncCardHeight(host)
				return true
			}
			if !host.pointerDragged {
				a.selectID(host.card.ID)
				if host.pressClicks >= 2 && a.config().Card.DoubleClickPaste {
					a.pasteCard(host)
				}
			}
			host.pointerDragged = false
			a.syncCardHeight(host)
			return true
		default:
			return false
		}
	})
}
