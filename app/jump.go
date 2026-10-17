package app

import (
	"math"
	"strings"

	"github.com/0froq/snip/config"
	"github.com/egoist/mygo/ui"
)

// BadgeDiameter is the jump-badge window size. It is even so the badge center
// lands on the card's top-right corner in integer screen coordinates.
const BadgeDiameter = 28

const whichKeyMargin = 16

// frame is a screen rectangle in top-left coordinates.
type frame struct {
	X, Y, W, H int
}

// jumpKeyPool is the label alphabet: digits 1-9, then 0, then letters, then
// other single keys. Bound leader follow-ups are removed before assignment.
var jumpKeyPool = []string{
	"1", "2", "3", "4", "5", "6", "7", "8", "9", "0",
	"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m",
	"n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",
	";", "'", ",", ".", "/", "[", "]", "\\", "-", "=", "`",
}

var hintKeys = map[string]string{
	"toggle":   "hint_toggle",
	"next":     "hint_next",
	"prev":     "hint_prev",
	"paste":    "hint_paste",
	"delete":   "hint_delete",
	"pin":      "hint_pin",
	"settings": "hint_settings",
}

type whichKeyItem struct {
	Key  string
	Name string
}

// BadgeFrame is the badge window whose center is the card's top-right corner.
// The bottom-left quarter of an even-sized badge lies on the card. The other
// three quarters sit outside, above the top edge and to the right of the card.
func BadgeFrame(card frame, diameter int) frame {
	d := evenDiameter(diameter)
	return frame{
		X: card.X + card.W - d/2,
		Y: card.Y - d/2,
		W: d,
		H: d,
	}
}

func evenDiameter(d int) int {
	if d < 2 {
		d = 2
	}
	if d%2 != 0 {
		d++
	}
	return d
}

func normalizeJumpKey(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func boundLeaderKeys(bindings config.ActionBindings) map[string]struct{} {
	out := map[string]struct{}{}
	for _, item := range bindings.NamedBindings() {
		if k := normalizeJumpKey(item.Binding.Leader); k != "" {
			out[k] = struct{}{}
		}
	}
	return out
}

// assignJumpLabels gives each id one label, in id order. Single keys are used
// while every id fits in the free pool. Past that, every label is two keys
// drawn from that same pool. Labels that match a bound follow-up are skipped.
// Entries stay empty once the pool is exhausted.
func assignJumpLabels(ids []string, bound map[string]struct{}, pool []string) []string {
	out := make([]string, len(ids))
	if len(ids) == 0 {
		return out
	}
	norm := map[string]struct{}{}
	for k := range bound {
		if n := normalizeJumpKey(k); n != "" {
			norm[n] = struct{}{}
		}
	}
	free := freeJumpKeys(norm, pool)
	if len(free) == 0 {
		return out
	}
	if len(ids) <= len(free) {
		copy(out, free[:len(ids)])
		return out
	}
	n := 0
	for _, a := range free {
		for _, b := range free {
			label := a + b
			if _, skip := norm[label]; skip {
				continue
			}
			if n >= len(out) {
				return out
			}
			out[n] = label
			n++
		}
	}
	return out
}

func freeJumpKeys(bound map[string]struct{}, pool []string) []string {
	var free []string
	seen := map[string]struct{}{}
	for _, k := range pool {
		k = normalizeJumpKey(k)
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		if _, skip := bound[k]; skip {
			continue
		}
		free = append(free, k)
	}
	return free
}

// placeWhichKey puts a window of the given size inside screen, inset by margin,
// in one of the four corners.
func placeWhichKey(screen frame, corner string, winW, winH, margin int) (int, int) {
	if margin < 0 {
		margin = 0
	}
	switch corner {
	case config.WhichKeyTopLeft:
		return screen.X + margin, screen.Y + margin
	case config.WhichKeyTopRight:
		return screen.X + screen.W - winW - margin, screen.Y + margin
	case config.WhichKeyBottomLeft:
		return screen.X + margin, screen.Y + screen.H - winH - margin
	case config.WhichKeyBottomRight:
		return screen.X + screen.W - winW - margin, screen.Y + screen.H - winH - margin
	default:
		panic("unhandled which-key corner: " + corner)
	}
}

func whichKeyItems(cfg config.Config) []whichKeyItem {
	var items []whichKeyItem
	for _, item := range cfg.Bindings.NamedBindings() {
		key := strings.TrimSpace(item.Binding.Leader)
		if key == "" {
			continue
		}
		items = append(items, whichKeyItem{Key: key, Name: actionHint(cfg.Language, item.Name)})
	}
	return items
}

func actionHint(lang, action string) string {
	key := action
	if h, ok := hintKeys[action]; ok {
		key = h
	}
	return tr(lang, key)
}

func whichKeySize(items []whichKeyItem, font ui.Font) (int, int) {
	if len(items) == 0 {
		return 0, 0
	}
	const padX, padY, gap, rowGap = 12, 10, 10, 4
	var maxW, lineH float32
	for _, it := range items {
		kw, kh := ui.ShapeText(it.Key, font, 0).Size()
		nw, nh := ui.ShapeText(it.Name, font, 0).Size()
		if row := kw + float32(gap) + nw; row > maxW {
			maxW = row
		}
		if h := max(kh, nh); h > lineH {
			lineH = h
		}
	}
	w := int(math.Ceil(float64(maxW))) + padX*2 + 8
	h := padY*2 + int(math.Ceil(float64(lineH)))*len(items) + rowGap*(len(items)-1) + 4
	if w < 120 {
		w = 120
	}
	return w, h
}

// badgeGlyphNudge is the Y delta for a label whose line box is already
// centered in the badge. A lining figure or capital sits on the baseline
// and fills about 0.70em, not the descent and not the accent space in the
// ascent, so the line-box center hangs low. Negative moves the glyph up.
// Line gap is split above and below the em box, so it does not move the
// baseline relative to that center.
func badgeGlyphNudge(m ui.FontMetrics, size float32) float32 {
	if m.Ascent <= 0 {
		return 0
	}
	if size <= 0 {
		size = m.Ascent + m.Descent
	}
	const capRatio = 0.70
	baselineBelowCenter := (m.Ascent - m.Descent) / 2
	return size*capRatio/2 - baselineBelowCenter
}
