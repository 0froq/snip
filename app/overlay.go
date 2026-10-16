package app

import (
	"time"

	"github.com/0froq/snip/internal/macplace"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type cutout struct {
	ok bool
	x  int
	y  int
	w  int
	h  int
}

func (a *App) showDim() {
	a.hideDim()
	op := a.config().Leader.DimOpacity
	if op <= 0 {
		op = 0.45
	}
	if op > 0.8 {
		op = 0.8
	}
	for _, d := range mygo.Screen.Displays() {
		b := d.Bounds
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:         "snip dim",
			X:             b.X,
			Y:             b.Y,
			Width:         b.Width,
			Height:        b.Height,
			Frameless:     true,
			Transparent:   true,
			Panel:         true,
			AlwaysOnTop:   true,
			DisableResize: true,
			Content:       ui.View(func(c *ui.Context) { a.dimView(c, b.Width, b.Height, op) }),
		})
		win.SetIgnoreMouseEvents(true)
		win.OnFocus(func() { win.ShowInactive() })
		win.ShowInactive()
		a.dimWins = append(a.dimWins, win)
	}
	go func() {
		done := make(chan struct{})
		var frame macplace.Rect
		var ok bool
		go func() {
			frame, ok = macplace.FocusedFrame()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(100 * time.Millisecond):
			ok = false
		}
		if !ok {
			return
		}
		mygo.RunOnMain(func() {
			if !a.leaderActive() {
				return
			}
			a.dimCut = cutout{ok: true, x: frame.X, y: frame.Y, w: frame.Width, h: frame.Height}
			for _, w := range a.dimWins {
				w.Invalidate()
			}
		})
	}()
}

func (a *App) hideDim() {
	for _, w := range a.dimWins {
		w.Close()
	}
	a.dimWins = nil
	a.dimCut = cutout{}
}

func (a *App) dimView(c *ui.Context, width, height int, opacity float64) {
	shade := ui.Hex("#1C1612").Alpha(float32(opacity))
	ui.Box(c).Fill().Children(func() {
		local, inside := a.localCutout(width, height, a.dimCut)
		if !a.dimCut.ok || !inside {
			ui.Box(c).Size(float32(width), float32(height)).Background(shade)
			return
		}
		x, y, w, h := float32(local.x), float32(local.y), float32(local.w), float32(local.h)
		fw, fh := float32(width), float32(height)
		if y > 0 {
			ui.Box(c).Absolute().Left(0).Top(0).Width(fw).Height(y).Background(shade)
		}
		if x > 0 {
			ui.Box(c).Absolute().Left(0).Top(y).Width(x).Height(h).Background(shade)
		}
		if right := fw - (x + w); right > 0 {
			ui.Box(c).Absolute().Left(x + w).Top(y).Width(right).Height(h).Background(shade)
		}
		if bottom := fh - (y + h); bottom > 0 {
			ui.Box(c).Absolute().Left(0).Top(y + h).Width(fw).Height(bottom).Background(shade)
		}
	})
}

func (a *App) localCutout(width, height int, cut cutout) (cutout, bool) {
	// The dim window is created at display bounds. Match the display whose
	// size equals this view when several monitors differ.
	for _, d := range mygo.Screen.Displays() {
		if d.Bounds.Width != width || d.Bounds.Height != height {
			continue
		}
		lx := cut.x - d.Bounds.X
		ly := cut.y - d.Bounds.Y
		if lx+cut.w < 0 || ly+cut.h < 0 || lx > width || ly > height {
			return cutout{}, false
		}
		return cutout{ok: true, x: lx, y: ly, w: cut.w, h: cut.h}, true
	}
	return cutout{ok: true, x: cut.x, y: cut.y, w: cut.w, h: cut.h}, true
}
