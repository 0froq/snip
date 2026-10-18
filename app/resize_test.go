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
	a := New(cfg, "")
	host := &cardUI{display: strings.Repeat("long text 你好 ", 100)}
	for _, w := range []int{120, 240, 480} {
		got := a.fitCardWidth(host, w)
		if got.Truncated || strings.Contains(got.Text, "…") || got.Text != host.display {
			t.Fatalf("width %d: text was clipped: %+v", w, got)
		}
		font := cfg.Card.UIFont(cfg.Card.EffectiveFontSize())
		inner := float32(got.Width - 2*cfg.Card.EffectivePaddingX() - 2*config.CardBorderPx())
		_, textH := ui.ShapeText(got.Text, font, inner).Size()
		wantH := int(math.Ceil(float64(textH))) + 2*cfg.Card.EffectivePaddingY() + 2*config.CardBorderPx()
		if wantH > cfg.Card.EffectiveMaxHeight() {
			wantH = cfg.Card.EffectiveMaxHeight()
		}
		if got.Height != wantH {
			t.Fatalf("width %d: height %d, measured height %d", w, got.Height, wantH)
		}
	}
	// A shorter note must immediately shrink, not retain the previous height.
	host.display = "small note"
	got := a.fitCardWidth(host, 240)
	if got.Height >= cfg.Card.MaxHeight || got.VisibleLines != 1 || got.Truncated || got.Scrolls {
		t.Fatalf("short note retains blank height: %+v", got)
	}
}

func TestFastBottomDragSnapsToWholeLinesAndPersists(t *testing.T) {
	cfg := config.Default()
	a := New(cfg, "")
	host := &cardUI{display: strings.Repeat("long text 你好 ", 80)}
	full := a.fitCardWidth(host, 360)
	if full.Truncated || full.Text != host.display || !full.Scrolls || full.Height != cfg.Card.EffectiveMaxHeight() {
		t.Fatalf("start layout: %+v", full)
	}
	one := a.layoutAt(host, full.Width, 1)
	if one.Height != cfg.Card.OneLineHeight(full.Width) || one.Text != host.display || !one.Scrolls {
		t.Fatalf("one-line viewport: %+v", one)
	}
	// A drag between one line and the text is kept. The card scrolls the
	// rest instead of ellipsizing to a whole line count.
	targetH := one.Height + 17
	if targetH >= full.Height {
		t.Fatal("fixture is not taller than one line")
	}
	w, h := resizeDragSize(full.Width, full.Height, edgeBottom, 0, targetH-full.Height)
	if h != targetH || w != full.Width {
		t.Fatalf("drag size = %d×%d, want %d×%d", w, h, full.Width, targetH)
	}
	got := a.fitCardSize(host, w, h)
	if got.Height != targetH || got.Text != host.display || !got.Scrolls || got.Width != full.Width {
		t.Fatalf("release kept the viewport: %+v", got)
	}
	if host.rendered != host.display || host.userH != targetH {
		t.Fatalf("published state: rendered=%q h=%d", host.rendered, host.userH)
	}
	// Moving the card syncs through fitCardWidth and must keep that viewport.
	moved := a.fitCardWidth(host, got.Width)
	if moved.Height != targetH || moved.Text != host.display || host.userH != targetH {
		t.Fatalf("move reset the viewport: %+v userH=%d", moved, host.userH)
	}

	// Native live-resize end passes the stuck window size with no pointer delta.
	host.userLines = 0
	host.rendered = ""
	native := a.fitCardSize(host, full.Width, targetH)
	if native.Height != targetH || native.Text != host.display || !native.Scrolls {
		t.Fatalf("native end: %+v", native)
	}
	if kept := a.fitCardWidth(host, native.Width); kept.Height != targetH || kept.Text != host.display {
		t.Fatalf("native size did not persist: %+v", kept)
	}
}

func TestFitCardSizeNeverShrinksBelowOneLine(t *testing.T) {
	cfg := config.Default()
	a := New(cfg, "")
	host := &cardUI{display: strings.Repeat("long text 你好 ", 40)}
	one := a.layoutAt(host, 360, 1)
	if one.Height != cfg.Card.OneLineHeight(360) || one.Text != host.display || !one.Scrolls {
		t.Fatalf("one-line box: %+v", one)
	}
	for _, h := range []int{0, -40, 1, one.Height - 1} {
		got := a.fitCardSize(host, 360, h)
		if got.Height != one.Height || got.Width != one.Width || got.Text != host.display || host.userH != one.Height {
			t.Fatalf("h=%d: got %+v userH=%d, want %+v", h, got, host.userH, one)
		}
	}
	floor := cfg.Card.WidthFloor(host.display)
	narrow := a.fitCardSize(host, -100, 0)
	if narrow.Width != floor || narrow.Width < 1 || narrow.Height < 1 || narrow.Text != host.display {
		t.Fatalf("width floor: %+v floor=%d", narrow, floor)
	}
}

func TestFastResizeFrameBeatsStaleLineCount(t *testing.T) {
	cfg := config.Default()
	a := New(cfg, "")
	host := &cardUI{display: strings.Repeat("long text 你好 ", 80)}
	full := a.fitCardWidth(host, 360)
	one := a.layoutAt(host, full.Width, 1)
	// The frame is already one line. The drawn string stays the whole note
	// and scrolls, including when the last publish was cleared.
	host.userLines = 0
	host.rendered = ""
	got := a.commitCardFrame(host, full.Width, one.Height)
	if got.Height != one.Height || got.Text != host.display || host.rendered != host.display || !got.Scrolls {
		t.Fatalf("frame commit clipped the string: %+v rendered=%q", got, host.rendered)
	}
	kept := a.fitCardWidth(host, got.Width)
	if kept.Height != one.Height || kept.Text != host.display || host.rendered != host.display {
		t.Fatalf("later width sync changed the text: %+v rendered=%q", kept, host.rendered)
	}
}

func TestRapidResizeSequenceSnapsToFinalFrame(t *testing.T) {
	cfg := config.Default()
	a := New(cfg, "")
	host := &cardUI{display: strings.Repeat("long text 你好 ", 80)}
	full := a.fitCardWidth(host, 360)
	one := a.layoutAt(host, full.Width, 1)
	twoH := one.Height + 24
	floor := cfg.Card.WidthFloor(host.display)
	host.beginResize()
	if cardOrderOutAllowed(host) {
		t.Fatal("a resize callback must not be allowed to order the card out")
	}
	frames := [][2]int{
		{full.Width, full.Height},
		{full.Width, full.Height - 3},
		{full.Width, twoH + 1},
		{full.Width, 0},
		{full.Width, -80},
		{full.Width, one.Height},
	}
	var got config.CardLayout
	for _, f := range frames {
		got = a.commitCardFrame(host, f[0], f[1])
		if got.Height < one.Height || got.VisibleLines < 1 || got.Width < floor || host.rendered != got.Text {
			t.Fatalf("frame %v produced %+v rendered=%q", f, got, host.rendered)
		}
	}
	host.endResize()
	if !cardOrderOutAllowed(host) {
		t.Fatal("resize depth leaked")
	}
	if got.Height != one.Height || got.Text != host.display || host.userH != one.Height || got.Truncated {
		t.Fatalf("final frame: %+v userH=%d", got, host.userH)
	}
	// A stale delta still describes the original card. The release uses the frame.
	dw, dh := resizeDragSize(full.Width, full.Height, edgeBottom, 0, -2)
	rw, rh := releaseFrame(full.Width, one.Height, dw, dh)
	if dh < twoH {
		t.Fatalf("setup delta = %dx%d, want it still multi-line", dw, dh)
	}
	if rw != full.Width || rh != one.Height {
		t.Fatalf("release = %dx%d, want frame %dx%d", rw, rh, full.Width, one.Height)
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
	a := New(cfg, "")
	host := &cardUI{display: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"}
	narrow := a.fitCardWidth(host, 120)
	wide := a.fitCardWidth(host, 480)
	if wide.Height >= narrow.Height {
		t.Fatalf("width increase must reduce content height: narrow %+v, wide %+v", narrow, wide)
	}
	a.cfg.Card.MaxWidth = 160
	one := a.cfg.Card.OneLineHeight(160)
	a.cfg.Card.MaxHeight = one
	got := a.fitCardWidth(host, host.userW)
	if got.Width != 160 || got.Height != one || !got.Scrolls || got.Truncated || got.Text != host.display {
		t.Fatalf("resize ignored current config: %+v", got)
	}
}
