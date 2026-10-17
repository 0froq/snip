//go:build !darwin

package macplace

import "github.com/0froq/snip/config"

// Supported is false off macOS.
const Supported = false

// Origin resolves card placement. Always returns ok=false on non-Darwin builds.
func Origin(_ config.Placement, _ int, _ int) (x, y int, ok bool) {
	return 0, 0, false
}

// SquareWindow is a no-op off macOS.
func SquareWindow(uintptr) {}

// PreventWindowFocus is a no-op off macOS.
func PreventWindowFocus(uintptr) {}

// DisplayWindow is a no-op off macOS.
func DisplayWindow(uintptr) {}

// ConstrainWindowResize is a no-op off macOS.
func ConstrainWindowResize(_ uintptr, _ func(int, int) (int, int)) func() { return func() {} }

func InLiveResize(handle uintptr) bool { return false }
