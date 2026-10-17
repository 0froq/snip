//go:build darwin

package macplace

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/ebitengine/purego/objc"
)

var focusTestSerial atomic.Uint64

func TestPreventWindowFocusIsInstanceLocalAndPreservesMouseHandling(t *testing.T) {
	withPool(func() {
		var mouseCalls int
		base, err := objc.RegisterClass(fmt.Sprintf("SnipFocusTestWindow_%d", focusTestSerial.Add(1)), objc.GetClass("NSObject"), nil, nil, []objc.MethodDef{
			{Cmd: objc.RegisterName("canBecomeKeyWindow"), Fn: func(id, objc.SEL) bool { return true }},
			{Cmd: objc.RegisterName("canBecomeMainWindow"), Fn: func(id, objc.SEL) bool { return true }},
			{Cmd: objc.RegisterName("mouseDown:"), Fn: func(id, objc.SEL, id) { mouseCalls++ }},
		})
		if err != nil {
			t.Fatal(err)
		}
		note := send(send(id(base), "alloc"), "init")
		defer send(note, "release")
		otherNote := send(send(id(base), "alloc"), "init")
		defer send(otherNote, "release")
		settings := send(send(id(base), "alloc"), "init")
		defer send(settings, "release")

		PreventWindowFocus(0) // Safe for missing/closed native handles.
		PreventWindowFocus(uintptr(note))
		noteClass := note.Class()
		PreventWindowFocus(uintptr(note))
		PreventWindowFocus(uintptr(otherNote))
		if noteClass == base || note.Class() != noteClass || otherNote.Class() != noteClass {
			t.Fatal("notes must reuse a per-base subclass, including after repeated configuration")
		}
		for _, selector := range []string{"canBecomeKeyWindow", "canBecomeMainWindow"} {
			// Send through the actual Objective-C IMP, not the Go callback.
			if send(note, selector) != 0 || send(otherNote, selector) != 0 {
				t.Errorf("note %s must be false", selector)
			}
			if send(settings, selector) == 0 || settings.Class() != base {
				t.Errorf("Settings/shared base %s must remain unchanged", selector)
			}
		}
		send(note, "mouseDown:", 0)
		if mouseCalls != 1 {
			t.Fatal("note subclass must preserve inherited mouse handling")
		}
	})
}
