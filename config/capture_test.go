package config

import "testing"

func TestCaptureAccepts(t *testing.T) {
	zed := "dev.zed.Zed"
	notes := "com.apple.Notes"

	all := Capture{Mode: CaptureAll, Blocklist: []string{zed}}
	if !all.Accepts(zed) || !all.Accepts(notes) || !all.Accepts("") {
		t.Fatal("all mode keeps every source")
	}

	block := Capture{Mode: "Blocklist", Blocklist: []string{" " + zed + " "}}
	if block.Accepts(zed) || block.Accepts("DEV.ZED.ZED") {
		t.Fatal("blocklist kept a listed app")
	}
	if !block.Accepts(notes) || !block.Accepts("") {
		t.Fatal("blocklist dropped an unlisted source")
	}

	allow := Capture{Mode: CaptureAllowlist, Allowlist: []string{notes}}
	if !allow.Accepts(notes) || !allow.Accepts("com.apple.notes") {
		t.Fatal("allowlist dropped a listed app")
	}
	if allow.Accepts(zed) || allow.Accepts("") {
		t.Fatal("allowlist kept an unlisted source")
	}

	if !(Capture{Mode: "elsewhere"}).Accepts(zed) {
		t.Fatal("unknown mode should keep copies")
	}
	if _, err := NormalizeCaptureMode("elsewhere"); err == nil {
		t.Fatal("settings should reject an unknown mode")
	}
}

func TestCaptureFormRoundTrip(t *testing.T) {
	src := Default()
	src.Capture.Mode = CaptureBlocklist
	src.Capture.Blocklist = []string{"dev.zed.Zed", " dev.zed.Zed", ""}
	src.Capture.Allowlist = []string{"com.apple.Notes"}
	got, err := FormFrom(src).Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if got.Capture.Mode != CaptureBlocklist {
		t.Fatalf("mode %q", got.Capture.Mode)
	}
	if len(got.Capture.Blocklist) != 1 || got.Capture.Blocklist[0] != "dev.zed.Zed" {
		t.Fatalf("blocklist %#v", got.Capture.Blocklist)
	}
	if !got.Capture.Accepts("com.apple.Notes") || got.Capture.Accepts("dev.zed.Zed") {
		t.Fatalf("filter %+v", got.Capture)
	}
}
