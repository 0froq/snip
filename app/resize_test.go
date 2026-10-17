package app

import (
	"math"
	"strings"
	"testing"

	"github.com/0froq/snip/config"
	"github.com/egoist/mygo/ui"
)

func TestFitCardWidthBoundsAndStoredSize(t *testing.T) {
	cfg := config.Default()
	a := New(cfg, "")
	host := &cardUI{display: strings.Repeat("a long note ", 30)}
	for _, w := range []int{-100, 0, 80, 200, 10000} {
		got := a.fitCardWidth(host, w)
		wantW := max(cfg.Card.WidthFloor(host.display), min(w, cfg.Card.EffectiveMaxWidth()))
		want := cfg.Card.Layout(host.display, wantW, 0)
		if got != want {
			t.Fatalf("width %d: got %+v, want %+v", w, got, want)
		}
		if host.userW != want.Width || host.userH != want.Height || host.rendered != want.Text {
			t.Fatalf("stored geometry/text not updated: %+v", host)
		}
	}
	// The existing content-bound floor must not stretch a short note.
	host.display = "hi"
	got := a.fitCardWidth(host, 1)
	if got.Width != cfg.Card.EstimateWidth("hi") || got.Width >= cfg.Card.EffectiveMinWidth() {
		t.Fatalf("short note stretched: %+v", got)
	}
}

func TestFitCardWidthMeasuresRenderedTextWithoutBlankHeight(t *testing.T) {
	cfg := config.Default()
	cfg.Card.MaxLines = 4
	a := New(cfg, "")
	host := &cardUI{display: strings.Repeat("long text 你好 ", 100)}
	for _, w := range []int{120, 240, 480} {
		got := a.fitCardWidth(host, w)
		if !got.Truncated || !strings.Contains(got.Text, "\n…\n") || got.VisibleLines != 4 {
			t.Fatalf("width %d: expected ellipsis: %+v", w, got)
		}
		font := cfg.Card.UIFont(cfg.Card.EffectiveFontSize())
		inner := float32(got.Width - 2*cfg.Card.EffectivePaddingX() - 2*config.CardBorderPx())
		_, textH := ui.ShapeText(got.Text, font, inner).Size()
		wantH := int(math.Ceil(float64(textH))) + 2*cfg.Card.EffectivePaddingY() + 2*config.CardBorderPx()
		if got.Height != wantH {
			t.Fatalf("width %d: height %d, exact content height %d", w, got.Height, wantH)
		}
	}
	// A shorter note must immediately shrink, not retain the previous height.
	host.display = "small note"
	got := a.fitCardWidth(host, 240)
	if got.Height >= cfg.Card.MaxHeight || got.VisibleLines != 1 || got.Truncated {
		t.Fatalf("short note retains blank height: %+v", got)
	}
}

func TestFastBottomDragSnapsToWholeLinesAndPersists(t *testing.T) {
	cfg := config.Default()
	cfg.Card.MaxLines = 4
	a := New(cfg, "")
	host := &cardUI{display: strings.Repeat("long text 你好 ", 80)}
	full := a.fitCardWidth(host, 360)
	if full.VisibleLines != 4 || host.userLines != 0 || !strings.Contains(full.Text, "\n…\n") {
		t.Fatalf("start layout: %+v userLines=%d", full, host.userLines)
	}
	one := a.layoutAt(host, full.Width, 1)
	two := a.layoutAt(host, full.Width, 2)
	if one.Height >= two.Height || one.VisibleLines != 1 || !one.Truncated {
		t.Fatalf("line heights: one %+v two %+v", one, two)
	}
	// A fast drag lands between 1 and 2 lines. The release must snap to the
	// largest whole count that still fits, with the middle-truncated text.
	targetH := one.Height + (two.Height-one.Height)/2
	w, h := resizeDragSize(full.Width, full.Height, edgeBottom, 0, targetH-full.Height)
	if h != targetH || w != full.Width {
		t.Fatalf("drag size = %d×%d, want %d×%d", w, h, full.Width, targetH)
	}
	got := a.fitCardSize(host, w, h)
	if got.VisibleLines != 1 || got.Height != one.Height || got.Text != one.Text || got.Width != one.Width {
		t.Fatalf("release snap: got %+v, want %+v", got, one)
	}
	if host.userLines != 1 || host.rendered != one.Text || host.userH != one.Height {
		t.Fatalf("published state: lines=%d rendered=%q h=%d", host.userLines, host.rendered, host.userH)
	}
	// Moving the card syncs through fitCardWidth. That must keep N, not grow
	// back to max_lines because userLines was left at 0.
	moved := a.fitCardWidth(host, got.Width)
	if moved.VisibleLines != 1 || moved.Height != one.Height || moved.Text != one.Text || host.userLines != 1 {
		t.Fatalf("move reset the line count: %+v userLines=%d", moved, host.userLines)
	}

	// Native live-resize end passes the stuck window size with no pointer delta.
	host.userLines = 0
	host.rendered = full.Text
	native := a.fitCardSize(host, full.Width, targetH)
	if native.VisibleLines != 1 || native.Height != one.Height || native.Text != one.Text || host.userLines != 1 {
		t.Fatalf("native end snap: %+v userLines=%d", native, host.userLines)
	}
	if kept := a.fitCardWidth(host, native.Width); kept.Height != one.Height || host.userLines != 1 {
		t.Fatalf("native snap did not persist: %+v userLines=%d", kept, host.userLines)
	}
}

func TestFitCardSizeNeverShrinksBelowOneLine(t *testing.T) {
	cfg := config.Default()
	a := New(cfg, "")
	host := &cardUI{display: strings.Repeat("long text 你好 ", 40)}
	one := a.layoutAt(host, 360, 1)
	if one.Height < 1 || one.VisibleLines != 1 {
		t.Fatalf("one-line box: %+v", one)
	}
	for _, h := range []int{0, -40, 1, one.Height - 1} {
		got := a.fitCardSize(host, 360, h)
		if got.VisibleLines != 1 || got.Height != one.Height || got.Width != one.Width || host.userLines != 1 {
			t.Fatalf("h=%d: got %+v userLines=%d, want %+v", h, got, host.userLines, one)
		}
	}
	floor := cfg.Card.WidthFloor(host.display)
	narrow := a.fitCardSize(host, -100, 0)
	if narrow.Width != floor || narrow.Width < 1 || narrow.VisibleLines != 1 || narrow.Height < 1 || host.userLines != 1 {
		t.Fatalf("width floor: %+v floor=%d userLines=%d", narrow, floor, host.userLines)
	}
}

func TestResizeDragSizeEdges(t *testing.T) {
	w, h := resizeDragSize(200, 100, edgeRight|edgeBottom, 15, -40)
	if w != 215 || h != 60 {
		t.Fatalf("right+bottom = %d×%d", w, h)
	}
	w, h = resizeDragSize(200, 100, edgeLeft, 15, 40)
	if w != 185 || h != 100 {
		t.Fatalf("left = %d×%d", w, h)
	}
}

func TestFitCardWidthReflowsAndHonorsConfigChanges(t *testing.T) {
	cfg := config.Default()
	cfg.Card.MaxLines = 10
	a := New(cfg, "")
	host := &cardUI{display: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"}
	narrow := a.fitCardWidth(host, 120)
	wide := a.fitCardWidth(host, 480)
	if wide.Height >= narrow.Height {
		t.Fatalf("width increase must reduce content height: narrow %+v, wide %+v", narrow, wide)
	}
	a.cfg.Card.MaxWidth = 160
	a.cfg.Card.MaxLines = 1
	got := a.fitCardWidth(host, host.userW)
	if got.Width != 160 || got.VisibleLines != 1 || !got.Truncated {
		t.Fatalf("resize ignored current config: %+v", got)
	}
}
