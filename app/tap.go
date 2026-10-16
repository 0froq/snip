package app

import "time"

// tapStep is one press in a double tap sequence.
// When a double action exists, the first press waits. A second press in time
// fires the double action. Otherwise the single action fires at once.
func tapStep(pending string, key, single, double string, elapsed, gap time.Duration) (fire string, next string, wait bool) {
	if pending == key && double != "" && elapsed <= gap {
		return double, "", false
	}
	if double == "" {
		return single, "", false
	}
	return "", key, true
}
