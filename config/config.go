package config

import (
	"os"
	"strings"
	"time"

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
	MinWidth    int    `toml:"min_width"`
	MaxWidth    int    `toml:"max_width"`
	Height      int    `toml:"height"`
	MaxHeight   int    `toml:"max_height"`
	LineHeight  int    `toml:"line_height"`
	MaxLines    int    `toml:"max_lines"`
	ActiveColor string `toml:"active_color"`
	PaddingX    int    `toml:"padding_x"`
	PaddingY    int    `toml:"padding_y"`
	// FontFamily is a comma-separated list. The first face that exists is
	// the UI font; later faces (PingFang SC) cover glyphs it lacks.
	FontFamily     string `toml:"font_family"`
	FontSize       int    `toml:"font_size"`
	CharWidth      int    `toml:"char_width"`
	CascadeStepX   int    `toml:"cascade_step_x"`
	CascadeStepY   int    `toml:"cascade_step_y"`
	CascadeOriginX int    `toml:"cascade_origin_x"`
	CascadeOriginY int    `toml:"cascade_origin_y"`
	// Placement is where new cards spawn: near-selection, near-cursor, or fixed.
	Placement string `toml:"placement"`
	// DoubleClickPaste pastes a card when it is double clicked.
	DoubleClickPaste bool `toml:"double_click_paste"`
}

// ActionBinding gives one action independent direct and leader bindings.
type ActionBinding struct {
	Direct       string `toml:"direct"`
	DirectDouble bool   `toml:"direct_double"`
	Leader       string `toml:"leader"`
	LeaderDouble bool   `toml:"leader_double"`
	Sticky       bool   `toml:"sticky"`
}

// ActionBindings contains every command that can be bound.
type ActionBindings struct {
	Toggle   ActionBinding `toml:"toggle"`
	Next     ActionBinding `toml:"next"`
	Prev     ActionBinding `toml:"prev"`
	Paste    ActionBinding `toml:"paste"`
	Delete   ActionBinding `toml:"delete"`
	Pin      ActionBinding `toml:"pin"`
	Settings ActionBinding `toml:"settings"`
}

// NamedBindings returns bindings in stable UI order.
func (b *ActionBindings) NamedBindings() []struct {
	Name    string
	Binding *ActionBinding
} {
	return []struct {
		Name    string
		Binding *ActionBinding
	}{
		{"toggle", &b.Toggle},
		{"next", &b.Next},
		{"prev", &b.Prev},
		{"paste", &b.Paste},
		{"delete", &b.Delete},
		{"pin", &b.Pin},
		{"settings", &b.Settings},
	}
}

// Leader controls the prefix and timing shared by leader bindings.
type Leader struct {
	Shortcut      string `toml:"shortcut"`
	TimeoutMs     int    `toml:"timeout_ms"`
	DoublePressMs int    `toml:"double_press_ms"`
	PinMs         int    `toml:"pin_ms"`
	// WhichKeyCorner is the screen corner for the leader which-key window:
	// top-left, top-right, bottom-left, or bottom-right.
	WhichKeyCorner string `toml:"which_key_corner"`
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
	ClipboardPollMs int            `toml:"clipboard_poll_ms"`
	MaxCards        int            `toml:"max_cards"`
	Window          Window         `toml:"window"`
	Card            Card           `toml:"card"`
	Bindings        ActionBindings `toml:"bindings"`
	Secrets         Secrets        `toml:"secrets"`
	Pasteboard      Pasteboard     `toml:"pasteboard"`
	Leader          Leader         `toml:"leader"`
	Motion          Motion         `toml:"motion"`
	Language        string         `toml:"language"`
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
			MinWidth:         120,
			MaxWidth:         480,
			Height:           44,
			MaxHeight:        220,
			LineHeight:       18,
			MaxLines:         4,
			ActiveColor:      "#C4784A",
			PaddingX:         10,
			PaddingY:         6,
			FontFamily:       DefaultFontFamily,
			FontSize:         DefaultFontSize,
			CharWidth:        7,
			CascadeStepX:     12,
			CascadeStepY:     10,
			CascadeOriginX:   24,
			CascadeOriginY:   24,
			DoubleClickPaste: true,
		},
		Bindings: ActionBindings{
			Toggle:   ActionBinding{Direct: "CmdOrCtrl+Shift+S", Leader: "s"},
			Next:     ActionBinding{Direct: "CmdOrCtrl+Shift+]", Leader: "j", Sticky: true},
			Prev:     ActionBinding{Direct: "CmdOrCtrl+Shift+[", Leader: "k", Sticky: true},
			Paste:    ActionBinding{Direct: "CmdOrCtrl+Shift+V", Leader: "v"},
			Delete:   ActionBinding{Direct: "CmdOrCtrl+Shift+Backspace", Leader: "d"},
			Pin:      ActionBinding{Direct: "CmdOrCtrl+Shift+F", Leader: "p", Sticky: true},
			Settings: ActionBinding{Direct: "CmdOrCtrl+,", Leader: ","},
		},
		Leader: Leader{
			Shortcut:       "CmdOrCtrl+Shift+Space",
			TimeoutMs:      1500,
			DoublePressMs:  400,
			PinMs:          900,
			WhichKeyCorner: WhichKeyBottomRight,
		},
		Motion:   Motion{Enabled: true, DurationMs: 160},
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

// EstimateSize returns a width and a height that fit the text, from one line up to max lines.
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

func (l Leader) PinDuration() time.Duration {
	if l.PinMs <= 0 {
		return 900 * time.Millisecond
	}
	return time.Duration(l.PinMs) * time.Millisecond
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
	return cfg, nil
}
