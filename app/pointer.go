package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const pointerSlop = 6.0

// cardPointerInput moves the card window on drag and selects on a tap.
// HandleInput must consume the pointer stream because DragWindow skips
// press tracking, so Clicked and release handlers never run.
func cardPointerInput(a *App, host *cardUI, win *mygo.Window, col ui.Element) ui.Element {
	return col.HandleInput(func(ev ui.InputEvent) bool {
		switch ev.Kind {
		case ui.InputPointerDown:
			if ev.Button != 0 {
				return false
			}
			host.pointerActive = true
			host.pointerDragged = false
			host.pointerX, host.pointerY = ev.X, ev.Y
			host.pointerWinX, host.pointerWinY = win.Position()
			return true
		case ui.InputPointerMove:
			if !host.pointerActive {
				return false
			}
			dx := ev.X - host.pointerX
			dy := ev.Y - host.pointerY
			if !host.pointerDragged {
				if dx*dx+dy*dy < pointerSlop*pointerSlop {
					return true
				}
				host.pointerDragged = true
			}
			win.SetPosition(host.pointerWinX+int(dx), host.pointerWinY+int(dy))
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
			return true
		default:
			return false
		}
	})
}
