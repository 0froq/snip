package config

import (
	"os"
	"strings"
	"time"

	"git.zacharyzhang.com/ZacharyZhang-NY/8bitMyGoUI/pixel"
	"github.com/BurntSushi/toml"
)

// Window holds floating-card window defaults (overridable per deployment).
type Window struct {
	AlwaysOnTop            bool `toml:"always_on_top"`
	VisibleOnAllWorkspaces bool `toml:"visible_on_all_workspaces"`
	ContentProtection      bool `toml:"content_protection"`
	// Frameless is omitted from config when true (default). Set false in TOML
	// only for advanced debugging; not exposed in Settings.
	Frameless *bool `toml:"frameless"`
	// Panel uses a non-normal window level (launcher-style) so tiling WMs
	// are less likely to manage snip cards as regular app windows.
	Panel *bool `toml:"panel"`
}

// Card controls the paper-strip snippet appearance.
type Card struct {
	MinWidth   int `toml:"min_width"`
	MaxWidth   int `toml:"max_width"`
	Height     int `toml:"height"`
	MaxHeight  int `toml:"max_height"`
	LineHeight int `toml:"line_height"`
	// MaxLines is accepted so older config files still load. Card text is
	// no longer clipped to a line count; MaxHeight is the viewport cap.
	MaxLines    int    `toml:"max_lines"`
	ActiveColor string `toml:"active_color"`
	PaddingX    int    `toml:"padding_x"`
	PaddingY    int    `toml:"padding_y"`
	// FontFamily is a comma-separated list. The first face that exists is
	// the UI font; later faces (PingFang SC) cover glyphs it lacks.
	FontFamily string `toml:"font_family"`
	FontSize   int    `toml:"font_size"`
	// FontKind is "pixel" for the bundled bitmap face, or "system".
	// Empty means system, so a config written before this field still loads.
	FontKind       string `toml:"card_font,omitempty"`
	CharWidth      int    `toml:"char_width"`
	CascadeStepX   int    `toml:"cascade_step_x"`
	CascadeStepY   int    `toml:"cascade_step_y"`
	CascadeOriginX int    `toml:"cascade_origin_x"`
	CascadeOriginY int    `toml:"cascade_origin_y"`
	// Placement is where new cards spawn: near-selection, near-cursor, or fixed.
	Placement string `toml:"placement"`
	// DoubleClickPaste pastes a card when it is double clicked.
	DoubleClickPaste bool `toml:"double_click_paste"`
	// ActiveOpacity is the selected card's window alpha. InactiveOpacity is
	// every other card. Both are 0.2–1. Zero means the default of 1.
	ActiveOpacity   float64 `toml:"active_opacity"`
	InactiveOpacity float64 `toml:"inactive_opacity"`
	// HoverHide drops a card to alpha 0 after the pointer has rested inside
	// its frame. HoverHideDelayMs is how long that rest lasts. Zero delay
	// means the default. HoverHideMargin expands the frame for restore, and
	// HoverHideRestoreDelayMs is how long the pointer must stay outside
	// that expanded frame. A negative margin means the default.
	HoverHide               bool `toml:"hover_hide"`
	HoverHideDelayMs        int  `toml:"hover_hide_delay_ms"`
	HoverHideMargin         int  `toml:"hover_hide_margin"`
	HoverHideRestoreDelayMs int  `toml:"hover_hide_restore_delay_ms"`
}

// Leader controls timing and the which-key panel. The leader key itself
// is a binding whose action is "leader".

type Leader struct {
	// Shortcut is the pre-list leader combo. Load copies it into the leader
	// binding. Saves omit it.
	Shortcut      string `toml:"shortcut,omitempty"`
	TimeoutMs     int    `toml:"timeout_ms"`
	DoublePressMs int    `toml:"double_press_ms"`
	// WhichKeyX and WhichKeyY are where the panel anchor sits, as a percent
	// of the screen work area (0..100). WhichKeyAlignX and WhichKeyAlignY
	// choose which point of the panel sits there: start (left/top), center,
	// or end (right/bottom).
	WhichKeyX      int    `toml:"which_key_x"`
	WhichKeyY      int    `toml:"which_key_y"`
	WhichKeyAlignX string `toml:"which_key_align_x"`
	WhichKeyAlignY string `toml:"which_key_align_y"`
}

// Motion controls appear and disappear animation.
type Motion struct {
	Enabled    bool `toml:"enabled"`
	DurationMs int  `toml:"duration_ms"`
}

// Secrets controls on-screen masking (not clipboard contents).
type Secrets struct {
	MaskEnabled     bool     `toml:"mask_enabled"`
	ExtraPatterns   []string `toml:"extra_patterns"`
	MinEntropyBits  float64  `toml:"min_entropy_bits"`
	MinSecretLength int      `toml:"min_secret_length"`
}

// Config is the full snip configuration.
type Config struct {
	ClipboardPollMs int         `toml:"clipboard_poll_ms"`
	MaxCards        int         `toml:"max_cards"`
	Window          Window      `toml:"window"`
	Card            Card        `toml:"card"`
	Bindings        BindingList `toml:"bindings"`
	Capture         Capture     `toml:"capture"`
	Secrets         Secrets     `toml:"secrets"`
	Pasteboard      Pasteboard  `toml:"pasteboard"`
	Leader          Leader      `toml:"leader"`
	Motion          Motion      `toml:"motion"`
	Theme           Theme       `toml:"theme"`
	Language        string      `toml:"language"`
}

// Default returns built-in defaults aligned with the prototype goals.
func Default() Config {
	return Config{
		ClipboardPollMs: 300,
		MaxCards:        20,
		Window: Window{
			AlwaysOnTop:            true,
			VisibleOnAllWorkspaces: true,
			ContentProtection:      true,
			Frameless:              boolPtr(true),
			Panel:                  boolPtr(true),
		},
		Card: Card{
			MinWidth:                120,
			MaxWidth:                480,
			Height:                  44,
			MaxHeight:               220,
			LineHeight:              18,
			MaxLines:                4,
			ActiveColor:             "#C4784A",
			PaddingX:                10,
			PaddingY:                6,
			FontFamily:              DefaultFontFamily,
			FontSize:                DefaultFontSize,
			CharWidth:               7,
			CascadeStepX:            12,
			CascadeStepY:            10,
			CascadeOriginX:          24,
			CascadeOriginY:          24,
			DoubleClickPaste:        true,
			ActiveOpacity:           1,
			InactiveOpacity:         1,
			HoverHide:               true,
			HoverHideDelayMs:        DefaultHoverHideDelayMs,
			HoverHideMargin:         DefaultHoverHideMargin,
			HoverHideRestoreDelayMs: DefaultHoverHideRestoreDelayMs,
		},
		Bindings: DefaultBindings(),
		Capture:  Capture{Mode: CaptureAll},
		Leader: Leader{
			TimeoutMs:      1500,
			DoublePressMs:  400,
			WhichKeyX:      100,
			WhichKeyY:      100,
			WhichKeyAlignX: AlignEnd,
			WhichKeyAlignY: AlignEnd,
		},
		Motion:   Motion{Enabled: true, DurationMs: 160},
		Theme:    Theme{Palette: PaletteAuto},
		Language: "en",
		Secrets: Secrets{
			MaskEnabled:     true,
			MinEntropyBits:  3.5,
			MinSecretLength: 16,
		},
		Pasteboard: DefaultPasteboard(),
	}
}

// EffectiveMaxCards returns the card cap (default 20 when unset or invalid).
func (c Config) EffectiveMaxCards() int {
	if c.MaxCards <= 0 {
		return 20
	}
	return c.MaxCards
}

// EstimateWidth is the initial slip width: measured text plus padding, capped
// at max width. min_width does not stretch a short note.
func (c Card) EstimateWidth(display string) int {
	est := c.naturalSlipWidth(display)
	if est > c.EffectiveMaxWidth() {
		return c.EffectiveMaxWidth()
	}
	return est
}

// WidthFloor is the narrowest width a drag may use. A note already narrower
// than min_width keeps that measured width so the window is not forced wider.
func (c Card) WidthFloor(display string) int {
	floor := c.EffectiveMinWidth()
	natural := c.EstimateWidth(display)
	if natural < floor {
		return natural
	}
	return floor
}

func (c Card) EffectiveMinWidth() int {
	if c.MinWidth <= 0 {
		return 120
	}
	return c.MinWidth
}

func (c Card) EffectiveMaxWidth() int {
	if c.MaxWidth <= 0 {
		return 480
	}
	return c.MaxWidth
}

func (c Card) EffectiveHeight() int {
	if c.Height <= 0 {
		return 44
	}
	return c.Height
}

func (c Card) EffectiveMaxHeight() int {
	if c.MaxHeight <= 0 {
		return 220
	}
	return c.MaxHeight
}

func (c Card) EffectiveLineHeight() int {
	if c.LineHeight <= 0 {
		return c.EffectiveFontSize() + 6
	}
	return c.LineHeight
}

// EstimateSize returns the natural width and height. Height is the measured
// text capped at max height; the card scrolls once the text is taller.
func (c Card) EstimateSize(display string) (int, int) {
	width := c.EstimateWidth(display)
	return width, c.HeightForWidth(display, width)
}

// HeightForWidth returns the window height needed to show display at the given slip width.
func (c Card) HeightForWidth(display string, width int) int {
	if width <= 0 {
		width = c.EstimateWidth(display)
	}
	if width > c.EffectiveMaxWidth() {
		width = c.EffectiveMaxWidth()
	}
	_, h := c.shapedHeight(display, width)
	return h
}

// EffectiveMaxLines reports the deprecated line cap. Layout does not clip
// to it. A missing or non-positive value stays at the historical default
// so old files that set max_lines keep a readable number in settings.
func (c Card) EffectiveMaxLines() int {
	if c.MaxLines <= 0 {
		return 4
	}
	return c.MaxLines
}

func (c Card) EffectivePaddingX() int {
	if c.PaddingX <= 0 {
		return 10
	}
	return c.PaddingX
}

func (c Card) EffectivePaddingY() int {
	if c.PaddingY <= 0 {
		return 6
	}
	return c.PaddingY
}

// DefaultFontFamily is the system sans with a CJK sans fallback.
const DefaultFontFamily = "-apple-system, PingFang SC, Hiragino Sans GB, sans-serif"

// DefaultFontSize is the card, badge and which-key text size in points.
const DefaultFontSize = 14

func (c Card) EffectiveFontFamily() string {
	if strings.TrimSpace(c.FontFamily) == "" {
		return DefaultFontFamily
	}
	return strings.TrimSpace(c.FontFamily)
}

func (c Card) EffectiveFontSize() int {
	if c.FontSize <= 0 {
		return DefaultFontSize
	}
	return c.FontSize
}

const (
	// CardFontSystem draws cards with the system face named by font_family.
	CardFontSystem = "system"
	// CardFontPixel draws cards with the bundled 8-bit face.
	CardFontPixel = "pixel"
	// PixelDot is DIPs per bitmap dot, the same step the card chrome uses.
	PixelDot = 2
)

// EffectiveCardFont is system or pixel. Anything else, including an empty
// value, is system so an older file keeps its current cards.
func (c Card) EffectiveCardFont() string {
	switch strings.ToLower(strings.TrimSpace(c.FontKind)) {
	case CardFontPixel:
		return CardFontPixel
	case CardFontSystem, "":
		return CardFontSystem
	default:
		return CardFontSystem
	}
}

// PixelFont reports whether the card draws the bundled bitmap face.
func (c Card) PixelFont() bool {
	return c.EffectiveCardFont() == CardFontPixel
}

// PixelScale is how many times the bitmap is drawn for the configured
// point size. The face is pixel.LineHeight dots and each dot is PixelDot
// DIPs, so scale 1 is a 32-DIP line and scale 2 is 64. The kit's dot stops
// at 4, which is scale 2, so a larger point size stays there. A size
// below one line stays at 1: the bitmap has no smaller step.
func (c Card) PixelScale() int {
	return PixelScale(c.EffectiveFontSize())
}

// PixelScale maps a point size onto the bitmap step. See Card.PixelScale.
func PixelScale(fontSize int) int {
	if fontSize <= 0 {
		fontSize = DefaultFontSize
	}
	line := pixel.LineHeight * PixelDot
	if fontSize >= line+line/2 {
		return 2
	}
	return 1
}

const (
	// OpacityMin and OpacityMax bound a card window's alpha.
	OpacityMin = 0.2
	// OpacityMax is fully opaque.
	OpacityMax = 1.0
)

// EffectiveActiveOpacity is the selected card's alpha. An unset value is
// fully opaque so a config written before this key stays as it was.
func (c Card) EffectiveActiveOpacity() float64 {
	return clampOpacity(c.ActiveOpacity)
}

// EffectiveInactiveOpacity is the alpha of a card that is not selected.
func (c Card) EffectiveInactiveOpacity() float64 {
	return clampOpacity(c.InactiveOpacity)
}

// OpacityFor returns the window alpha for a selected or idle card.
func (c Card) OpacityFor(selected bool) float64 {
	if selected {
		return c.EffectiveActiveOpacity()
	}
	return c.EffectiveInactiveOpacity()
}

// clampOpacity maps an unset value to fully opaque and keeps every other
// value inside 0.2–1.
func clampOpacity(v float64) float64 {
	if v == 0 {
		return OpacityMax
	}
	if v < OpacityMin {
		return OpacityMin
	}
	if v > OpacityMax {
		return OpacityMax
	}
	return v
}

// DefaultHoverHideDelayMs is how long the pointer must stay inside a card
// before that card hides itself.
const DefaultHoverHideDelayMs = 1500

// EffectiveHoverHideDelayMs is the dwell time. An unset value keeps the
// default so an older config file does not hide cards immediately.
func (c Card) EffectiveHoverHideDelayMs() int {
	if c.HoverHideDelayMs <= 0 {
		return DefaultHoverHideDelayMs
	}
	return c.HoverHideDelayMs
}

// HoverHideDelay is the dwell time as a duration.
func (c Card) HoverHideDelay() time.Duration {
	return time.Duration(c.EffectiveHoverHideDelayMs()) * time.Millisecond
}

// DefaultHoverHideMargin is how far past the card, in points, the pointer
// must travel before a hidden card can come back.
const DefaultHoverHideMargin = 40

// DefaultHoverHideRestoreDelayMs is how long the pointer must stay outside
// the expanded frame before a hidden card fades back.
const DefaultHoverHideRestoreDelayMs = 2000

// EffectiveHoverHideMargin is the restore margin. Negative means the
// default. Zero is a real margin of zero, so a file can turn the buffer off.
func (c Card) EffectiveHoverHideMargin() int {
	if c.HoverHideMargin < 0 {
		return DefaultHoverHideMargin
	}
	return c.HoverHideMargin
}

// EffectiveHoverHideRestoreDelayMs is the outside wait. An unset value
// keeps the default.
func (c Card) EffectiveHoverHideRestoreDelayMs() int {
	if c.HoverHideRestoreDelayMs <= 0 {
		return DefaultHoverHideRestoreDelayMs
	}
	return c.HoverHideRestoreDelayMs
}

// HoverHideRestoreDelay is the outside wait as a duration.
func (c Card) HoverHideRestoreDelay() time.Duration {
	return time.Duration(c.EffectiveHoverHideRestoreDelayMs()) * time.Millisecond
}

func (c Card) EffectiveCharWidth() int {
	if c.CharWidth <= 0 {
		return 7
	}
	return c.CharWidth
}

func (l Leader) Timeout() time.Duration {
	if l.TimeoutMs <= 0 {
		return 1500 * time.Millisecond
	}
	return time.Duration(l.TimeoutMs) * time.Millisecond
}

func (l Leader) DoubleGap() time.Duration {
	if l.DoublePressMs <= 0 {
		return 400 * time.Millisecond
	}
	return time.Duration(l.DoublePressMs) * time.Millisecond
}

func (m Motion) Duration() time.Duration {
	if m.DurationMs <= 0 {
		return 160 * time.Millisecond
	}
	return time.Duration(m.DurationMs) * time.Millisecond
}

// PollInterval returns the clipboard polling interval.
func (c Config) PollInterval() time.Duration {
	if c.ClipboardPollMs <= 0 {
		return 300 * time.Millisecond
	}
	return time.Duration(c.ClipboardPollMs) * time.Millisecond
}

// Load reads a TOML file and merges it onto defaults.
func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	cfg.Leader.applyLegacyWhichKeyFile(data)
	cfg.finishLegacyLeader(data)
	return cfg, nil
}

// applyLegacyWhichKeyFile honors which_key_corner only when the file does
// not set the percent or align keys. Those replace the corner option.
func (l *Leader) applyLegacyWhichKeyFile(data []byte) {
	var raw struct {
		Leader struct {
			Corner string  `toml:"which_key_corner"`
			X      *int    `toml:"which_key_x"`
			Y      *int    `toml:"which_key_y"`
			AlignX *string `toml:"which_key_align_x"`
			AlignY *string `toml:"which_key_align_y"`
		} `toml:"leader"`
	}
	if err := toml.Unmarshal(data, &raw); err != nil || raw.Leader.Corner == "" {
		return
	}
	if raw.Leader.X != nil || raw.Leader.Y != nil || raw.Leader.AlignX != nil || raw.Leader.AlignY != nil {
		return
	}
	l.ApplyLegacyWhichKeyCorner(raw.Leader.Corner)
}
