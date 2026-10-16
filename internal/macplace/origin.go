package macplace

import (
	"time"

	"github.com/0froq/snip/config"
)

const originTimeout = 100 * time.Millisecond

// ResolveOrigin runs placement off the clipboard poll thread and invokes fn with the result.
// When AX is slow, the call returns after originTimeout and ok is false.
func ResolveOrigin(mode config.Placement, cardW, cardH int, fn func(x, y int, ok bool)) {
	if mode == config.PlacementFixed {
		fn(0, 0, false)
		return
	}
	go func() {
		done := make(chan struct{})
		var x, y int
		var ok bool
		go func() {
			x, y, ok = Origin(mode, cardW, cardH)
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(originTimeout):
			ok = false
		}
		fn(x, y, ok)
	}()
}
