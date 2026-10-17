package macplace

import "testing"

func TestWindowCornerRadius(t *testing.T) {
	if WindowCornerRadius != 6 {
		t.Fatalf("corner radius %v", WindowCornerRadius)
	}
}
