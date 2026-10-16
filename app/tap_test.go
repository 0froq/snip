package app

import (
	"testing"
	"time"
)

func TestTapStepSingle(t *testing.T) {
	fire, next, wait := tapStep("", "v", "paste", "", 0, time.Second)
	if fire != "paste" || next != "" || wait {
		t.Fatalf("%q %q %v", fire, next, wait)
	}
}

func TestTapStepWaitsThenDoubles(t *testing.T) {
	fire, next, wait := tapStep("", "v", "paste", "pin", 0, time.Second)
	if fire != "" || next != "v" || !wait {
		t.Fatalf("first %q %q %v", fire, next, wait)
	}
	fire, next, wait = tapStep("v", "v", "paste", "pin", 100*time.Millisecond, time.Second)
	if fire != "pin" || next != "" || wait {
		t.Fatalf("second %q %q %v", fire, next, wait)
	}
}
