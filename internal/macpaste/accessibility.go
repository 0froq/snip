package macpaste

import (
	"log"
	"sync"
)

// axPrompt opens the Accessibility pane on the first call and reports that
// later calls should only be logged.
type axPrompt struct {
	once sync.Once
}

func (p *axPrompt) allow(open func()) bool {
	opened := false
	p.once.Do(func() {
		opened = true
		if open != nil {
			open()
		}
	})
	return opened
}

var accessibilityPrompt axPrompt

// OpenAccessibilitySettings opens Privacy > Accessibility at most once per
// process. Later calls log and leave System Settings alone.
func OpenAccessibilitySettings() {
	if accessibilityPrompt.allow(openAccessibilityPane) {
		log.Printf("snip: accessibility permission missing; opening System Settings")
		return
	}
	log.Printf("snip: accessibility permission missing")
}
