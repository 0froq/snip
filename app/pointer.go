package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const pointerSlop = 6.0

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
