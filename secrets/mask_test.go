package secrets

import "testing"

func TestDisplayTextMasksKnownPatterns(t *testing.T) {
	cfg := Config{Enabled: true, MinEntropyBits: 10, MinSecretLength: 32}
	raw := "token=supersecretvalue123 and sk-abcdefghijklmnopqrstuvwxyz123456"
	out := DisplayText(raw, cfg)
	if out == raw {
		t.Fatalf("expected masking, got %q", out)
	}
	if !contains(out, "•") {
		t.Fatalf("expected bullets in %q", out)
	}
}

func TestDisplayTextDisabled(t *testing.T) {
	cfg := Config{Enabled: false}
	raw := "password=hunter2"
	if DisplayText(raw, cfg) != raw {
		t.Fatal("disabled should not mask")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
