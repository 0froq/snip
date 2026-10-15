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

// OpenAccessibilitySettings is a no-op off macOS.
func OpenAccessibilitySettings() {}

// errUnsupported is returned when macOS APIs are not available.
var errUnsupported = unsupportedError{}

type unsupportedError struct{}

func (unsupportedError) Error() string { return "macpaste: macOS only" }
