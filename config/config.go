package config

import (
	"os"
	"strings"
	"time"
	"unicode/utf8"

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
	MinWidth       int    `toml:"min_width"`
	MaxWidth       int    `toml:"max_width"`
	Height         int    `toml:"height"`
	MaxHeight      int    `toml:"max_height"`
	LineHeight     int    `toml:"line_height"`
	MaxLines       int    `toml:"max_lines"`
	ActiveColor    string `toml:"active_color"`
	PaddingX       int    `toml:"padding_x"`
	PaddingY       int    `toml:"padding_y"`
	FontSize       int    `toml:"font_size"`
	CharWidth      int    `toml:"char_width"`
	CascadeStepX   int    `toml:"cascade_step_x"`
	CascadeStepY   int    `toml:"cascade_step_y"`
	CascadeOriginX int    `toml:"cascade_origin_x"`
	CascadeOriginY int    `toml:"cascade_origin_y"`
	// Placement is where new cards spawn: near-selection, near-cursor, or fixed.
	Placement string `toml:"placement"`
}

// Shortcuts are global accelerator strings (MyGo syntax).
type Shortcuts struct {
	Toggle   string `toml:"toggle"`
	ShowAll  string `toml:"show_all"`
	HideAll  string `toml:"hide_all"`
	Next     string `toml:"next"`
	Prev     string `toml:"prev"`
	Paste    string `toml:"paste"`
	Delete   string `toml:"delete"`
	Pin      string `toml:"pin"`
	Settings string `toml:"settings"`
	// Direct registers the shortcuts above as global hotkeys.
	Direct bool `toml:"direct"`
}

// Leader is the vim style prefix and the single keys that follow it.
type Leader struct {
	Shortcut      string            `toml:"shortcut"`
	TimeoutMs     int               `toml:"timeout_ms"`
	Toggle        string            `toml:"toggle"`
	Next          string            `toml:"next"`
	Prev          string            `toml:"prev"`
	Paste         string            `toml:"paste"`
	Delete        string            `toml:"delete"`
	Pin           string            `toml:"pin"`
	Settings      string            `toml:"settings"`
	DoublePressMs int               `toml:"double_press_ms"`
	Double        map[string]string `toml:"double"`
	PinMs         int               `toml:"pin_ms"`
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
	ClipboardPollMs int        `toml:"clipboard_poll_ms"`
	MaxCards        int        `toml:"max_cards"`
	Window          Window     `toml:"window"`
	Card            Card       `toml:"card"`
	Shortcuts       Shortcuts  `toml:"shortcuts"`
	Secrets         Secrets    `toml:"secrets"`
	Pasteboard      Pasteboard `toml:"pasteboard"`
	Leader          Leader     `toml:"leader"`
	Motion          Motion     `toml:"motion"`
	Language        string     `toml:"language"`
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
			MinWidth:       120,
			MaxWidth:       480,
			Height:         44,
			MaxHeight:      220,
			LineHeight:     18,
			MaxLines:       2,
			ActiveColor:    "#C4784A",
			PaddingX:       10,
			PaddingY:       6,
			FontSize:       12,
			CharWidth:      7,
			CascadeStepX:   12,
			CascadeStepY:   10,
			CascadeOriginX: 24,
			CascadeOriginY: 24,
		},
		Shortcuts: Shortcuts{
			Toggle:   "CmdOrCtrl+Shift+S",
			Next:     "CmdOrCtrl+Shift+]",
			Prev:     "CmdOrCtrl+Shift+[",
			Paste:    "CmdOrCtrl+Shift+V",
			Delete:   "CmdOrCtrl+Shift+Backspace",
			Pin:      "CmdOrCtrl+Shift+F",
			Settings: "CmdOrCtrl+,",
			Direct:   true,
		},
		Leader: Leader{
			Shortcut:      "CmdOrCtrl+Shift+Space",
			TimeoutMs:     1500,
			Toggle:        "s",
			Next:          "j",
			Prev:          "k",
			Paste:         "v",
			Delete:        "d",
			Pin:           "p",
			Settings:      ",",
			DoublePressMs: 400,
			Double:        map[string]string{},
			PinMs:         900,
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

// EstimateCardWidth returns content width for a snippet (clamped min/max).
func (c Card) EstimateWidth(display string) int {
	minW := c.EffectiveMinWidth()
	maxW := c.EffectiveMaxWidth()
	charW := c.EffectiveCharWidth()
	pad := c.EffectivePaddingX()
	runes := utf8.RuneCountInString(display)
	est := runes*charW + 2*pad
	if est < minW {
		return minW
	}
	if est > maxW {
		return maxW
	}
	return est
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

// HeightForWidth returns the window height needed to show display at the given width.
func (c Card) HeightForWidth(display string, width int) int {
	minW := c.EffectiveMinWidth()
	maxW := c.EffectiveMaxWidth()
	if width < minW {
		width = minW
	}
	if width > maxW {
		width = maxW
	}
	inner := width - 2*c.EffectivePaddingX()
	charW := c.EffectiveCharWidth()
	if inner < charW {
		inner = charW
	}
	per := inner / charW
	if per < 1 {
		per = 1
	}
	runes := utf8.RuneCountInString(display)
	if runes < 1 {
		runes = 1
	}
	lines := (runes + per - 1) / per
	maxLines := c.EffectiveMaxLines()
	if lines > maxLines {
		lines = maxLines
	}
	if lines < 1 {
		lines = 1
	}
	minH := c.EffectiveLineHeight() + 2*c.EffectivePaddingY()
	h := lines*c.EffectiveLineHeight() + 2*c.EffectivePaddingY()
	if h < minH {
		h = minH
	}
	if h > c.EffectiveMaxHeight() {
		h = c.EffectiveMaxHeight()
	}
	return h
}

func (c Card) EffectiveMaxLines() int {
	if c.MaxLines <= 0 {
		return 2
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

func (c Card) EffectiveFontSize() int {
	if c.FontSize <= 0 {
		return 12
	}
	return c.FontSize
}

func (c Card) EffectiveCharWidth() int {
	if c.CharWidth <= 0 {
		return 7
	}
	return c.CharWidth
}

// EffectiveToggle is the show and hide shortcut. Older configs used show_all.
func (s Shortcuts) EffectiveToggle() string {
	if strings.TrimSpace(s.Toggle) != "" {
		return strings.TrimSpace(s.Toggle)
	}
	if strings.TrimSpace(s.ShowAll) != "" {
		return strings.TrimSpace(s.ShowAll)
	}
	return "CmdOrCtrl+Shift+S"
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
