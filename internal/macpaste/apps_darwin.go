//go:build darwin

package macpaste

import (
	"sort"
	"strings"
)

// RunningApp is a launched application with a bundle id.
type RunningApp struct {
	Name     string
	BundleID string
}

// FrontBundleID is the frontmost app's bundle id when a pasteboard change
// is observed. An app without a bundle id returns empty.
func FrontBundleID() string {
	load()
	var bundle string
	withPool(func() {
		app := send(send(class("NSWorkspace"), "sharedWorkspace"), "frontmostApplication")
		if app == 0 {
			return
		}
		bundle = strings.TrimSpace(goString(send(app, "bundleIdentifier")))
	})
	return bundle
}

// RunningApps lists launched apps that have a bundle id, by name.
func RunningApps() []RunningApp {
	load()
	var out []RunningApp
	withPool(func() {
		apps := send(send(class("NSWorkspace"), "sharedWorkspace"), "runningApplications")
		if apps == 0 {
			return
		}
		n := sendInt(apps, "count")
		seen := map[string]struct{}{}
		for i := 0; i < n; i++ {
			app := send(apps, "objectAtIndex:", uintptr(i))
			if app == 0 {
				continue
			}
			id := strings.TrimSpace(goString(send(app, "bundleIdentifier")))
			if id == "" {
				continue
			}
			key := strings.ToLower(id)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			name := strings.TrimSpace(goString(send(app, "localizedName")))
			if name == "" {
				name = id
			}
			out = append(out, RunningApp{Name: name, BundleID: id})
		}
	})
	sort.Slice(out, func(i, j int) bool {
		ni, nj := strings.ToLower(out[i].Name), strings.ToLower(out[j].Name)
		if ni == nj {
			return out[i].BundleID < out[j].BundleID
		}
		return ni < nj
	})
	return out
}
