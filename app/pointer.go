package app

import (
	"github.com/0froq/snip/config"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const pointerSlop = 6.0

const resizeEdgePx = 10

const (
	edgeE = 1
	edgeW = 2
	edgeS = 4
)

// snapLines moves start by whole lines of linePx. dy is downward in pixels.
func snapLines(start, dy, linePx, minLines, maxLines int) int {
	if linePx < 1 {
		linePx = 1
	}
	if minLines < 1 {
		minLines = 1
	}
	n := start + dy/linePx
	if n < minLines {
		n = minLines
	}
	if maxLines < minLines {
		maxLines = minLines
	}
	if n > maxLines {
		n = maxLines
	}
	return n
}

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
			host.pressClicks = ev.Clicks
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

func (a *App) cardResizeEdges(c *ui.Context, host *cardUI, win *mygo.Window) {
	a.cardResizeEdge(c, host, win, edgeW, ui.CursorResizeW, func(b ui.Element) ui.Element {
		return b.Absolute().Left(0).Top(0).Bottom(resizeEdgePx).Width(resizeEdgePx)
	})
	a.cardResizeEdge(c, host, win, edgeE, ui.CursorResizeE, func(b ui.Element) ui.Element {
		return b.Absolute().Right(0).Top(0).Bottom(resizeEdgePx).Width(resizeEdgePx)
	})
	a.cardResizeEdge(c, host, win, edgeS, ui.CursorResizeS, func(b ui.Element) ui.Element {
		return b.Absolute().Left(resizeEdgePx).Right(resizeEdgePx).Bottom(0).Height(resizeEdgePx)
	})
	a.cardResizeEdge(c, host, win, edgeW|edgeS, ui.CursorResizeNESW, func(b ui.Element) ui.Element {
		return b.Absolute().Left(0).Bottom(0).Width(resizeEdgePx).Height(resizeEdgePx)
	})
	a.cardResizeEdge(c, host, win, edgeE|edgeS, ui.CursorResizeNWSE, func(b ui.Element) ui.Element {
		return b.Absolute().Right(0).Bottom(0).Width(resizeEdgePx).Height(resizeEdgePx)
	})
}

func (a *App) cardResizeEdge(c *ui.Context, host *cardUI, win *mygo.Window, edge int, cursor ui.Cursor, place func(ui.Element) ui.Element) {
	grip := place(ui.Box(c).Cursor(cursor))
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
			host.pressScreenX, host.pressScreenY = sp.X, sp.Y
			host.resizeStartW = ww
			host.resizeStartX = wx
			host.resizeStartLines = a.slipLines(host, ww)
			return true
		case ui.InputPointerMove:
			if !host.pointerActive || host.resizeEdge != edge {
				return false
			}
			a.applySlipResize(host, win)
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

func (a *App) applySlipResize(host *cardUI, win *mygo.Window) {
	cardCfg := a.config().Card
	sp := mygo.Screen.CursorScreenPoint()
	dx := sp.X - host.pressScreenX
	dy := sp.Y - host.pressScreenY
	w := host.resizeStartW
	x := host.resizeStartX
	edge := host.resizeEdge
	if edge&edgeE != 0 || edge&edgeW != 0 {
		dir := 1
		if edge&edgeW != 0 {
			dir = -1
		}
		var shift int
		w, shift = resizeWidth(host.resizeStartW, dx, dir, cardCfg.EffectiveMinWidth(), cardCfg.EffectiveMaxWidth())
		x = host.resizeStartX + shift
	}
	lines := host.resizeStartLines
	if edge&edgeS != 0 {
		full := cardCfg.Layout(host.display, w, 1<<20).FullLines
		linePx := cardCfg.Layout("M", w, 1).Height - 2*cardCfg.EffectivePaddingY() - 2*config.CardBorderPx()
		lines = snapLines(host.resizeStartLines, dy, linePx, 1, full)
		host.lines = lines
	} else {
		lines = a.slipLines(host, w)
	}
	layout := cardCfg.Layout(host.display, w, lines)
	host.rendered = layout.Text
	h := layout.Height
	_, y := win.Position()
	win.SetSize(w, h)
	win.SetPosition(x, y)
}
