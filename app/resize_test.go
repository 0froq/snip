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
		font := ui.Font{Family: "Georgia", Size: float32(cfg.Card.EffectiveFontSize())}
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
