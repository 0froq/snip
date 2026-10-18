package macplace

import "testing"

func TestCardMenuOmitsEditWhenSensitive(t *testing.T) {
	titles := CardMenuTitles{Edit: "Edit", Copy: "Copy", Delete: "Delete", Restore: "Restore original"}
	full := CardMenuItems(true, titles)
	if len(full) != 4 || full[0].Action != CardMenuEdit || full[1].Action != CardMenuCopy || full[2].Action != CardMenuDelete || full[3].Action != CardMenuRestore {
		t.Fatalf("menu %+v", full)
	}
	locked := CardMenuItems(false, titles)
	if len(locked) != 3 || locked[0].Action == CardMenuEdit {
		t.Fatalf("sensitive menu %+v", locked)
	}
	for i, item := range locked {
		if item.Action != full[i+1].Action || item.Title != full[i+1].Title {
			t.Fatalf("item %d: %+v", i, item)
		}
	}
}

func TestEditCommitKeys(t *testing.T) {
	if !EditCommit(EditKeyEscape, false) || !EditCommit(EditKeyEscape, true) {
		t.Fatal("escape commits")
	}
	if !EditCommit(EditKeyReturn, true) {
		t.Fatal("command-return commits")
	}
	if EditCommit(EditKeyReturn, false) || EditCommit(EditKeyOther, true) {
		t.Fatal("plain return and other keys stay in the editor")
	}
}

func TestScrollHitLeavesLeftClickToTheCard(t *testing.T) {
	if ScrollHit(false, 1, false) {
		t.Fatal("left mouse down should pass through")
	}
	if !ScrollHit(false, eventScrollWheel, false) || !ScrollHit(false, eventRightMouseDown, false) {
		t.Fatal("scroll and right click stay on the text")
	}
	if !ScrollHit(false, 1, true) {
		t.Fatal("the scroller keeps the left click")
	}
	if !ScrollHit(true, 1, false) {
		t.Fatal("editing takes the left click")
	}
}

func TestTextContentFrameClearsTheResizeBand(t *testing.T) {
	x, y, w, h, ok := TextContentFrame(200, 80, 10, 6, 1, 10, 0)
	if !ok || x != 11 || y != 10 || w != 200-22 || h != 80-20 {
		t.Fatalf("frame %v %v %v %v ok=%v", x, y, w, h, ok)
	}
	if _, _, _, _, ok := TextContentFrame(20, 80, 10, 6, 1, 10, 0); ok {
		t.Fatal("a window narrower than the insets is not a text frame")
	}
}

func TestFontFaceSkipsGenerics(t *testing.T) {
	name, system := FontFace("-apple-system, PingFang SC, Hiragino Sans GB, sans-serif")
	if system || name != "PingFang SC" {
		t.Fatalf("name %q system %v", name, system)
	}
	if _, system := FontFace("-apple-system, sans-serif"); !system {
		t.Fatal("generics use the system font")
	}
	if name, system := FontFace("  Menlo  "); system || name != "Menlo" {
		t.Fatalf("name %q system %v", name, system)
	}
}
