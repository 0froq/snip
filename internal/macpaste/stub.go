//go:build !darwin

package macpaste

// Supported is false on non-macOS builds.
const Supported = false

// ChangeCount returns 0 when pasteboard access is unavailable.
func ChangeCount() int { return 0 }

// ReadPlainText returns empty on non-macOS.
func ReadPlainText() string { return "" }

// AccessibilityTrusted reports whether paste simulation can run.
func AccessibilityTrusted(prompt bool) bool { return false }

// PostCommandV is a no-op off macOS.
func PostCommandV() error { return errUnsupported }

// WatchFrontmost is a no-op off macOS.
func WatchFrontmost() {}

// PasteToPrevious is a no-op off macOS.
func PasteToPrevious() error { return errUnsupported }

// openAccessibilityPane is a no-op off macOS.
func openAccessibilityPane() {}

// errUnsupported is returned when macOS APIs are not available.
var errUnsupported = unsupportedError{}

type unsupportedError struct{}

func (unsupportedError) Error() string { return "macpaste: macOS only" }
