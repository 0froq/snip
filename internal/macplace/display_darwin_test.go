//go:build darwin

package macplace

import (
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/ebitengine/purego/objc"
)

var displayTestSerial atomic.Uint64

func TestDisplayWindowRefreshesDescendantsWithoutFocus(t *testing.T) {
	withPool(func() {
		var calls []string
		names := map[id]string{}
		children := map[id]id{}
		viewClass, err := objc.RegisterClass(fmt.Sprintf("SnipDisplayTestView_%d", displayTestSerial.Add(1)), objc.GetClass("NSObject"), nil, nil, []objc.MethodDef{
			{Cmd: objc.RegisterName("layoutSubtreeIfNeeded"), Fn: func(self id, _ objc.SEL) {
				calls = append(calls, "layout "+names[self])
			}},
			{Cmd: objc.RegisterName("setNeedsDisplay:"), Fn: func(self id, _ objc.SEL, needs bool) {
				if !needs {
					t.Error("must request display, not clear the dirty flag")
				}
				calls = append(calls, "dirty "+names[self])
			}},
			{Cmd: objc.RegisterName("subviews"), Fn: func(self id, _ objc.SEL) id {
				return children[self]
			}},
			{Cmd: objc.RegisterName("displayIfNeeded"), Fn: func(self id, _ objc.SEL) {
				calls = append(calls, "display "+names[self])
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		newView := func(name string) id {
			v := send(send(id(viewClass), "alloc"), "init")
			names[v] = name
			children[v] = send(class("NSMutableArray"), "array")
			return v
		}
		root := newView("content")
		defer send(root, "release")
		container := newView("container")
		defer send(container, "release")
		surface := newView("surface")
		defer send(surface, "release")
		sibling := newView("sibling")
		defer send(sibling, "release")
		send(children[root], "addObject:", uintptr(container))
		send(children[root], "addObject:", uintptr(sibling))
		send(children[container], "addObject:", uintptr(surface))

		var content id = root
		windowClass, err := objc.RegisterClass(fmt.Sprintf("SnipDisplayTestWindow_%d", displayTestSerial.Add(1)), objc.GetClass("NSObject"), nil, nil, []objc.MethodDef{{
			Cmd: objc.RegisterName("contentView"), Fn: func(id, objc.SEL) id { return content },
		}})
		if err != nil {
			t.Fatal(err)
		}
		window := send(send(id(windowClass), "alloc"), "init")
		defer send(window, "release")
		// These NSObject stand-ins deliberately implement no ordering, key,
		// first-responder, or activation methods. Any focus attempt fails.
		DisplayWindow(0)
		for range 2 {
			calls = nil
			DisplayWindow(uintptr(window))
			want := []string{"layout content", "dirty content", "dirty container", "dirty surface", "dirty sibling", "display content"}
			if !reflect.DeepEqual(calls, want) {
				t.Fatalf("display calls = %v, want %v", calls, want)
			}
		}
		calls = nil
		content = 0 // Closed/window without content is harmless.
		DisplayWindow(uintptr(window))
		if len(calls) != 0 {
			t.Fatalf("displayed a missing content view: %v", calls)
		}
	})
}
