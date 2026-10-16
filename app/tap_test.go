package app

import (
	"testing"
	"time"
)

func TestPressOnlyRequiresSecondPress(t *testing.T) {
	var fired []string
	var gate tapGate
	gate.pressOnly("direct:paste", "paste", true, time.Second, func(action string) {
		fired = append(fired, action)
	})
	if len(fired) != 0 {
		t.Fatalf("first press fired %v", fired)
	}
	gate.pressOnly("direct:paste", "paste", true, time.Second, func(action string) {
		fired = append(fired, action)
	})
	if len(fired) != 1 || fired[0] != "paste" {
		t.Fatalf("second press fired %v", fired)
	}
}

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
