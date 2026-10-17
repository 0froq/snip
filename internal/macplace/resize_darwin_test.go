//go:build darwin

package macplace

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

var resizeTestSerial atomic.Uint64

// Exercise the actual Objective-C IMP ABI rather than calling the Go callback
// directly. In particular, an NSSize passed/returned through integer registers
// or a hidden result pointer would fail this test on arm64.
func TestResizeDelegateNSSizeABIAndLifetime(t *testing.T) {
	withPool(func() {
		serial := resizeTestSerial.Add(1)
		var original id
		var inheritedCalls int
		base, err := objc.RegisterClass(fmt.Sprintf("SnipResizeTestDelegate_%d", serial), objc.GetClass("NSObject"), nil, nil, []objc.MethodDef{{
			Cmd: objc.RegisterName("windowDidResize:"),
			Fn: func(self id, _ objc.SEL, _ id) {
				if self != original {
					t.Error("inherited method lost original delegate identity")
				}
				inheritedCalls++
			},
		}})
		if err != nil {
			t.Fatal(err)
		}
		original = send(send(id(base), "alloc"), "init")
		defer send(original, "release")
		delegate := original
		willResizeSEL := objc.RegisterName("windowWillResize:toSize:")
		var cachedWillResize bool
		windowClass, err := objc.RegisterClass(fmt.Sprintf("SnipResizeTestWindow_%d", serial), objc.GetClass("NSObject"), nil, nil, []objc.MethodDef{
			{Cmd: objc.RegisterName("delegate"), Fn: func(id, objc.SEL) id { return delegate }},
			{Cmd: objc.RegisterName("setDelegate:"), Fn: func(_ id, _ objc.SEL, d id) {
				delegate = d
				cachedWillResize = send(d, "respondsToSelector:", uintptr(willResizeSEL)) != 0
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		window := send(send(id(windowClass), "alloc"), "init")
		defer send(window, "release")

		before := len(resizeSizes)
		makeSizer := func() func(int) (int, int) {
			offset := new(int)
			*offset = 7
			return func(w int) (int, int) { return w + *offset, w/3 + 23 }
		}
		cleanup := ConstrainWindowResize(uintptr(window), makeSizer())
		defer cleanup()
		if delegate != original || !cachedWillResize {
			t.Fatal("delegate identity or NSWindow optional-method cache was not preserved")
		}
		if send(original, "retainCount") != 2 {
			t.Fatal("resize delegate must own an explicit retain")
		}
		runtime.GC()
		var willResize func(id, objc.SEL, id, NSSize) NSSize
		purego.RegisterFunc(&willResize, msgSendAddr)
		for _, proposed := range []NSSize{{120.25, 999}, {263.75, 1}, {480, 480}} {
			got := willResize(original, willResizeSEL, window, proposed)
			w := int(proposed.Width + 0.5)
			want := NSSize{float64(w + 7), float64(w/3 + 23)}
			if got != want {
				t.Errorf("NSSize ABI: proposed %+v, got %+v, want %+v", proposed, got, want)
			}
		}
		send(original, "windowDidResize:", 0)
		if inheritedCalls != 1 {
			t.Fatal("MyGo-style delegate notifications must still reach inherited methods")
		}
		cleanup()
		cleanup() // Safe even if an owner also explicitly tears down.
		if len(resizeSizes) != before || original.Class() != base || send(original, "retainCount") != 1 {
			t.Fatal("cleanup must unroot Go state, restore the class and balance the retain")
		}
	})
}
