//go:build !darwin

package macpaste

// ReadMarkers returns no markers off macOS.
func ReadMarkers() Markers { return Markers{} }
