package macpaste

import "errors"

// ErrAccessibility means simulated paste cannot run until the app is trusted.
var ErrAccessibility = errors.New("macpaste: accessibility permission required for simulated paste")
