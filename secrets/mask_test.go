package secrets

import "testing"

func TestDisplayTextMasksKnownPatterns(t *testing.T) {
	cfg := Config{Enabled: true, MinEntropyBits: 10, MinSecretLength: 32}
	raw := "token=supersecretvalue123 and sk-abcdefghijklmnopqrstuvwxyz123456"
	out := DisplayText(raw, cfg, false)
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
	if DisplayText(raw, cfg, false) != raw {
		t.Fatal("disabled should not mask")
	}
}

func TestIsSensitiveMatchesDisplayMasking(t *testing.T) {
	cfg := Config{Enabled: true, MinEntropyBits: 10, MinSecretLength: 32}
	raw := "token=supersecretvalue123"
	if !IsSensitive(raw, cfg, false) {
		t.Fatal("pattern match should be sensitive")
	}
	if IsSensitive("hello", cfg, false) {
		t.Fatal("plain text is editable")
	}
	off := Config{Enabled: false, HonorPasteboardConcealed: true}
	if IsSensitive("hunter2", off, false) {
		t.Fatal("masking off and not concealed")
	}
	if !IsSensitive("hunter2", off, true) {
		t.Fatal("concealed pasteboard is sensitive")
	}
	off.HonorPasteboardConcealed = false
	if IsSensitive("hunter2", off, true) {
		t.Fatal("concealed is ignored when honor is off")
	}
	if IsSensitive("", cfg, true) {
		t.Fatal("empty is not sensitive")
	}
}

func TestDisplayTextPasteboardConcealed(t *testing.T) {
	cfg := Config{Enabled: false, HonorPasteboardConcealed: true}
	raw := "hunter2"
	out := DisplayText(raw, cfg, true)
	if out == raw || !contains(out, "•") {
		t.Fatalf("concealed should mask even when pattern masking is off: %q", out)
	}
	cfg.HonorPasteboardConcealed = false
	if DisplayText(raw, cfg, true) != raw {
		t.Fatal("honor off should show concealed text")
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
