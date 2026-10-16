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
