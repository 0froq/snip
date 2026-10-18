package config

import (
	"errors"
	"strings"
)

var errCaptureMode = errors.New("capture mode must be all, blocklist, or allowlist")

const (
	// CaptureAll keeps a card for every app.
	CaptureAll = "all"
	// CaptureBlocklist drops copies whose frontmost app is listed.
	CaptureBlocklist = "blocklist"
	// CaptureAllowlist keeps a card only when the frontmost app is listed.
	CaptureAllowlist = "allowlist"
)

// Capture chooses which frontmost apps may open a card.
type Capture struct {
	Mode      string   `toml:"mode"`
	Blocklist []string `toml:"blocklist"`
	Allowlist []string `toml:"allowlist"`
}

// EffectiveMode is all, blocklist, or allowlist. Anything else is all.
func (c Capture) EffectiveMode() string {
	switch strings.ToLower(strings.TrimSpace(c.Mode)) {
	case "", CaptureAll:
		return CaptureAll
	case CaptureBlocklist:
		return CaptureBlocklist
	case CaptureAllowlist:
		return CaptureAllowlist
	default:
		return CaptureAll
	}
}

// Accepts reports whether a pasteboard change from bundleID may open a card.
// bundleID is the frontmost app at the time the change is seen. An empty id
// is kept in blocklist mode and dropped in allowlist mode.
func (c Capture) Accepts(bundleID string) bool {
	id := strings.TrimSpace(bundleID)
	switch c.EffectiveMode() {
	case CaptureAll:
		return true
	case CaptureBlocklist:
		return !bundleListed(c.Blocklist, id)
	case CaptureAllowlist:
		return bundleListed(c.Allowlist, id)
	default:
		return true
	}
}

func bundleListed(list []string, id string) bool {
	if id == "" {
		return false
	}
	for _, item := range list {
		if strings.EqualFold(strings.TrimSpace(item), id) {
			return true
		}
	}
	return false
}

// NormalizeCaptureMode returns a known mode or an error.
func NormalizeCaptureMode(mode string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case CaptureAll:
		return CaptureAll, nil
	case CaptureBlocklist:
		return CaptureBlocklist, nil
	case CaptureAllowlist:
		return CaptureAllowlist, nil
	default:
		return "", errCaptureMode
	}
}

// CleanBundleIDs trims, drops blanks, and keeps the first of each id.
func CleanBundleIDs(ids []string) []string {
	if len(ids) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(ids))
	seen := map[string]struct{}{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		key := strings.ToLower(id)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, id)
	}
	return out
}
