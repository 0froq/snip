package config

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Binding is one shortcut for an action. Keys is either a global combo
// ("cmd+shift+v") or a leader combo ("leader d"), pressed after the leader key.
// Stay keeps leader mode open after a leader combo runs.
type Binding struct {
	Action string `toml:"action"`
	Keys   string `toml:"keys"`
	Stay   bool   `toml:"stay"`
}

// ParsedBinding is a shortcut string after parsing.
type ParsedBinding struct {
	// Leader is true when the combo is pressed after the leader key.
	Leader bool
	// Chord is the canonical accelerator, without the leader prefix.
	Chord string
}

// BindingIssue is a problem with one shortcut string.
type BindingIssue struct {
	// Code is empty, unknown, conflict, or leader.
	Code  string
	Token string
}

func (e *BindingIssue) Error() string {
	if e == nil {
		return ""
	}
	if e.Token == "" {
		return e.Code
	}
	return e.Code + ": " + e.Token
}

// HasLeaderPrefix reports whether s starts a leader combo.
func HasLeaderPrefix(s string) bool {
	fields := strings.Fields(s)
	return len(fields) > 0 && strings.EqualFold(fields[0], "leader")
}

// ParseBinding parses a shortcut string.
// An empty string, an unknown key, and a leader prefix with no key are errors.
func ParseBinding(s string) (ParsedBinding, error) {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ParsedBinding{}, &BindingIssue{Code: "empty"}
	}
	leader := strings.EqualFold(fields[0], "leader")
	raw := s
	if leader {
		raw = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), fields[0]))
		if strings.TrimSpace(raw) == "" {
			return ParsedBinding{}, &BindingIssue{Code: "empty"}
		}
	}
	chord, err := canonicalChord(raw)
	if err != nil {
		return ParsedBinding{}, err
	}
	return ParsedBinding{Leader: leader, Chord: chord}, nil
}

func canonicalChord(s string) (string, error) {
	parts := splitChord(strings.TrimSpace(s))
	var mods modSet
	for i, raw := range parts {
		p := strings.TrimSpace(raw)
		if p == "" {
			return "", &BindingIssue{Code: "unknown", Token: s}
		}
		last := i == len(parts)-1
		if m, ok := chordModifier(p); ok && !last {
			mods |= m
			continue
		}
		if !last {
			return "", &BindingIssue{Code: "unknown", Token: p}
		}
		key, err := canonicalKey(p)
		if err != nil {
			return "", err
		}
		return mods.String() + key, nil
	}
	return "", &BindingIssue{Code: "empty"}
}

type modSet uint8

const (
	modCmd modSet = 1 << iota
	modCtrl
	modAlt
	modShift
)

func (m modSet) String() string {
	var parts []string
	if m&modCmd != 0 {
		parts = append(parts, "CmdOrCtrl")
	}
	if m&modCtrl != 0 {
		parts = append(parts, "Ctrl")
	}
	if m&modAlt != 0 {
		parts = append(parts, "Alt")
	}
	if m&modShift != 0 {
		parts = append(parts, "Shift")
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "+") + "+"
}

func chordModifier(p string) (modSet, bool) {
	switch strings.ToLower(p) {
	case "command", "cmd", "super", "meta", "win", "commandorcontrol", "cmdorctrl":
		return modCmd, true
	case "control", "ctrl":
		return modCtrl, true
	case "alt", "option", "altgr":
		return modAlt, true
	case "shift":
		return modShift, true
	default:
		return 0, false
	}
}

func canonicalKey(p string) (string, error) {
	if utf8.RuneCountInString(p) == 1 {
		return strings.ToUpper(p), nil
	}
	lower := strings.ToLower(p)
	if k, ok := namedKeys[lower]; ok {
		return k, nil
	}
	if len(lower) >= 2 && lower[0] == 'f' {
		var n int
		if _, err := fmt.Sscanf(lower[1:], "%d", &n); err == nil && n >= 1 && n <= 24 && fmt.Sprint(n) == lower[1:] {
			return fmt.Sprintf("F%d", n), nil
		}
	}
	if strings.HasPrefix(lower, "num") && len(lower) == 4 && lower[3] >= '0' && lower[3] <= '9' {
		return "Num" + lower[3:], nil
	}
	return "", &BindingIssue{Code: "unknown", Token: p}
}

// namedKeys matches the shortcut names mygo registers.
var namedKeys = map[string]string{
	"plus":               "+",
	"space":              "Space",
	"tab":                "Tab",
	"capslock":           "CapsLock",
	"numlock":            "NumLock",
	"scrolllock":         "ScrollLock",
	"backspace":          "Backspace",
	"delete":             "Delete",
	"del":                "Delete",
	"insert":             "Insert",
	"ins":                "Insert",
	"return":             "Enter",
	"enter":              "Enter",
	"up":                 "Up",
	"down":               "Down",
	"left":               "Left",
	"right":              "Right",
	"home":               "Home",
	"end":                "End",
	"pageup":             "PageUp",
	"pagedown":           "PageDown",
	"escape":             "Escape",
	"esc":                "Escape",
	"volumeup":           "VolumeUp",
	"volumedown":         "VolumeDown",
	"volumemute":         "VolumeMute",
	"medianexttrack":     "MediaNextTrack",
	"mediaprevioustrack": "MediaPreviousTrack",
	"mediastop":          "MediaStop",
	"mediaplaypause":     "MediaPlayPause",
	"printscreen":        "PrintScreen",
	"numdec":             "NumDec",
	"numadd":             "NumAdd",
	"numsub":             "NumSub",
	"nummult":            "NumMult",
	"numdiv":             "NumDiv",
}

// splitChord splits on "+" and keeps a trailing "+" as the plus key.
func splitChord(s string) []string {
	parts := strings.Split(s, "+")
	if strings.HasSuffix(s, "++") {
		parts = append(parts[:len(parts)-2], "+")
	} else if s == "+" {
		parts = []string{"+"}
	}
	return parts
}
