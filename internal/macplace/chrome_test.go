package macplace

import "testing"

func TestWindowMaskIsSquare(t *testing.T) {
	if WindowCornerRadius != 0 {
		t.Fatalf("corner radius %v", WindowCornerRadius)
	}
}
