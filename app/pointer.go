package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const pointerSlop = 6.0

const resizeEdgePx = 10

// resizeWidth applies a horizontal edge drag. edge is 1 for the right edge
// and -1 for the left. xShift is how far the left edge moves.
func resizeWidth(startW, dx, edge, minW, maxW int) (width, xShift int) {
	if edge == 0 {
		edge = 1
	}
	w := startW + dx*edge
	if w < minW {
		w = minW
	}
	if w > maxW {
		w = maxW
	}
	if edge < 0 {
		xShift = startW - w
	}
	return w, xShift
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
			host.pointerActive = true
			host.pointerDragged = false
			sp := mygo.Screen.CursorScreenPoint()
			wx, wy := win.Position()
			host.pressScreenX, host.pressScreenY = sp.X, sp.Y
			host.dragOffX = sp.X - wx
			host.dragOffY = sp.Y - wy
			return true
		case ui.InputPointerMove:
			if !host.pointerActive {
				return false
			}
			sp := mygo.Screen.CursorScreenPoint()
			dx := sp.X - host.pressScreenX
			dy := sp.Y - host.pressScreenY
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
			if !host.pointerDragged {
				a.selectID(host.card.ID)
			}
			host.pointerDragged = false
			a.syncCardHeight(host)
			return true
		default:
			return false
		}
	})
}

func (a *App) cardResizeEdges(c *ui.Context, host *cardUI, win *mygo.Window) {
	a.cardResizeEdge(c, host, win, 1)
	a.cardResizeEdge(c, host, win, -1)
}

func (a *App) cardResizeEdge(c *ui.Context, host *cardUI, win *mygo.Window, edge int) {
	grip := ui.Box(c).Absolute().Top(0).Bottom(0).Width(resizeEdgePx).Cursor(ui.CursorResizeEW)
	if edge > 0 {
		grip = grip.Right(0)
	} else {
		grip = grip.Left(0)
	}
	grip.HandleInput(func(ev ui.InputEvent) bool {
		switch ev.Kind {
		case ui.InputPointerDown:
			if ev.Button != 0 {
				return false
			}
			sp := mygo.Screen.CursorScreenPoint()
			wx, _ := win.Position()
			ww, _ := win.Size()
			host.pointerActive = true
			host.pointerDragged = true
			host.resizeEdge = edge
			host.pressScreenX = sp.X
			host.resizeStartW = ww
			host.resizeStartX = wx
			return true
		case ui.InputPointerMove:
			if !host.pointerActive || host.resizeEdge != edge {
				return false
			}
			sp := mygo.Screen.CursorScreenPoint()
			cardCfg := a.config().Card
			newW, shift := resizeWidth(host.resizeStartW, sp.X-host.pressScreenX, edge, cardCfg.EffectiveMinWidth(), cardCfg.EffectiveMaxWidth())
			newH := cardCfg.HeightForWidth(host.display, newW)
			_, wy := win.Position()
			win.SetPosition(host.resizeStartX+shift, wy)
			win.SetSize(newW, newH)
			return true
		case ui.InputPointerUp:
			if ev.Button != 0 || !host.pointerActive || host.resizeEdge != edge {
				return false
			}
			host.pointerActive = false
			host.pointerDragged = false
			host.resizeEdge = 0
			a.syncCardHeight(host)
			return true
		default:
			return false
		}
	})
}
