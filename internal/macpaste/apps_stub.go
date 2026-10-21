//go:build !darwin

package macpaste

// RunningApp is a launched application with a bundle id.
type RunningApp struct {
	Name     string
	BundleID string
}

// FrontBundleID is empty off macOS.
func FrontBundleID() string { return "" }

// RunningApps is empty off macOS.
func RunningApps() []RunningApp { return nil }
