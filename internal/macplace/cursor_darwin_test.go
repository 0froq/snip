//go:build darwin

package macplace

import (
	"testing"

	"github.com/ebitengine/purego/objc"
)

func TestFrameResizeCursors(t *testing.T) {
	UseFrameResizeCursors()
	cls := objc.ID(objc.GetClass("NSCursor"))
	if !responds(cls, "frameResizeCursorFromPosition:inDirections:") {
		t.Skip("pre-macOS 15")
	}
	if send(cls, "resizeUpDownCursor") != send(cls, "frameResizeCursorFromPosition:inDirections:", 1<<2, 3) {
		t.Fatal("up/down cursor not replaced")
	}
	if send(cls, "resizeLeftRightCursor") != send(cls, "frameResizeCursorFromPosition:inDirections:", 1<<3, 3) {
		t.Fatal("left/right cursor not replaced")
	}
}
