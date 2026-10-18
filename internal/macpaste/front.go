package macpaste

import (
	"errors"
	"sync"
)

// ErrNoPreviousApp means snip has not yet seen another frontmost app.
var ErrNoPreviousApp = errors.New("macpaste: no previous application")

// ErrPreviousAppInactive means the previous app did not become frontmost
// before the paste keystroke.
var ErrPreviousAppInactive = errors.New("macpaste: previous application did not become frontmost")

// frontHistory is the last frontmost process that is not this one.
// NSWorkspaceDidActivateApplicationNotification feeds it; snip's own pid
// is ignored so a click on a card does not erase the app to paste into.
type frontHistory struct {
	mu   sync.Mutex
	self int
	pid  int
	ok   bool
}

func (h *frontHistory) setSelf(pid int) {
	if pid <= 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.self = pid
	if h.pid == h.self {
		h.pid = 0
		h.ok = false
	}
}

func (h *frontHistory) note(pid int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if pid <= 0 || pid == h.self {
		return
	}
	h.pid = pid
	h.ok = true
}

func (h *frontHistory) previous() (int, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.ok || h.pid <= 0 || h.pid == h.self {
		return 0, false
	}
	return h.pid, true
}

var fronts frontHistory
