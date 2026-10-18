package app

import (
	"testing"

	"github.com/0froq/snip/config"
)

func TestStayKeepsLeaderMode(t *testing.T) {
	ended := false
	finishLeaderAction(true, func() { ended = true })
	if ended {
		t.Fatal("stay left leader mode")
	}
	finishLeaderAction(false, func() { ended = true })
	if !ended {
		t.Fatal("a binding without stay left leader mode open")
	}

	var stayed bool
	for _, row := range config.Default().Bindings {
		if row.Action != "next" || !row.Stay {
			continue
		}
		parsed, err := config.ParseBinding(row.Keys)
		if err != nil || !parsed.Leader {
			t.Fatalf("next stay row %q %v", row.Keys, err)
		}
		stayed = true
	}
	if !stayed {
		t.Fatal("default next leader binding does not stay")
	}
}
