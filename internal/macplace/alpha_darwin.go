//go:build darwin

package macplace

import (
	"sync"
	"time"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

var (
	alphaAnimOnce sync.Once
	// setAlphaFloat sends a CGFloat or NSTimeInterval. On arm64 those live
	// in a floating register, so this cannot go through SyscallN.
	setAlphaFloat func(id, objc.SEL, float64)
)

func ensureAlphaAnim() {
	load()
	alphaAnimOnce.Do(func() {
		purego.RegisterFunc(&setAlphaFloat, msgSendAddr)
	})
}

// AnimateWindowAlpha fades alphaValue to alpha over d via the window's
// animator. A later call replaces an in-flight fade and continues from the
// value on screen. d <= 0 commits alpha immediately, which is how hide-all
// and show stop a hover fade before setting their own alpha.
func AnimateWindowAlpha(handle uintptr, alpha float64, d time.Duration) {
	if handle == 0 {
		return
	}
	ensureAlphaAnim()
	if alpha < 0 {
		alpha = 0
	} else if alpha > 1 {
		alpha = 1
	}
	sec := d.Seconds()
	if sec < 0 {
		sec = 0
	}
	window := id(handle)
	ctx := class("NSAnimationContext")
	send(ctx, "beginGrouping")
	current := send(ctx, "currentContext")
	if current != 0 {
		setAlphaFloat(current, sel("setDuration:"), sec)
	}
	animator := send(window, "animator")
	if animator != 0 {
		setAlphaFloat(animator, sel("setAlphaValue:"), alpha)
	}
	send(ctx, "endGrouping")
}

// StopWindowAlphaAnimation ends an animator fade at the model alpha so the
// fade cannot write a later value over hide-all or show.
func StopWindowAlphaAnimation(handle uintptr) {
	if handle == 0 {
		return
	}
	AnimateWindowAlpha(handle, readWindowAlpha(id(handle)), 0)
}
