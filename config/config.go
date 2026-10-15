package config

import (
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

// Window holds floating-card window defaults (overridable per deployment).
type Window struct {
	AlwaysOnTop            bool `toml:"always_on_top"`
	VisibleOnAllWorkspaces bool `toml:"visible_on_all_workspaces"`
	ContentProtection      bool `toml:"content_protection"`
	Frameless              bool `toml:"frameless"`
	Width                  int  `toml:"width"`
	Height                 int  `toml:"height"`
}

// Shortcuts are global accelerator strings (MyGo syntax).
type Shortcuts struct {
	ShowAll   string `toml:"show_all"`
	HideAll   string `toml:"hide_all"`
	Next      string `toml:"next"`
	Prev      string `toml:"prev"`
	Paste     string `toml:"paste"`
	Delete    string `toml:"delete"`
}

// Secrets controls on-screen masking (not clipboard contents).
type Secrets struct {
	MaskEnabled      bool     `toml:"mask_enabled"`
	ExtraPatterns    []string `toml:"extra_patterns"`
	MinEntropyBits   float64  `toml:"min_entropy_bits"`
	MinSecretLength  int      `toml:"min_secret_length"`
}

// Config is the full snip configuration.
type Config struct {
	ClipboardPollMs int       `toml:"clipboard_poll_ms"`
	MaxCards        int       `toml:"max_cards"`
	Window          Window    `toml:"window"`
	Shortcuts       Shortcuts `toml:"shortcuts"`
	Secrets         Secrets   `toml:"secrets"`
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
			Frameless:              true,
			Width:                  320,
			Height:                 120,
		},
		Shortcuts: Shortcuts{
			ShowAll: "CmdOrCtrl+Shift+S",
			HideAll: "CmdOrCtrl+Shift+H",
			Next:    "CmdOrCtrl+Shift+]",
			Prev:    "CmdOrCtrl+Shift+[",
			Paste:   "CmdOrCtrl+Shift+V",
			Delete:  "CmdOrCtrl+Shift+Backspace",
		},
		Secrets: Secrets{
			MaskEnabled:     true,
			MinEntropyBits:  3.5,
			MinSecretLength: 16,
		},
	}
}

// EffectiveMaxCards returns the card cap (default 20 when unset or invalid).
func (c Config) EffectiveMaxCards() int {
	if c.MaxCards <= 0 {
		return 20
	}
	return c.MaxCards
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
