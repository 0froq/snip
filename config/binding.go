package config

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// DefaultBindings is the built-in shortcut list. The leader key is first.
func DefaultBindings() BindingList {
	return BindingList{
		{Action: "leader", Keys: "CmdOrCtrl+Shift+Space"},
		{Action: "toggle", Keys: "CmdOrCtrl+Shift+S"},
		{Action: "toggle", Keys: "leader s"},
		{Action: "next", Keys: "CmdOrCtrl+Shift+]"},
		{Action: "next", Keys: "leader j", Stay: true},
		{Action: "prev", Keys: "CmdOrCtrl+Shift+["},
		{Action: "prev", Keys: "leader k", Stay: true},
		{Action: "paste", Keys: "CmdOrCtrl+Shift+V"},
		{Action: "paste", Keys: "leader v"},
		{Action: "delete", Keys: "CmdOrCtrl+Shift+Backspace"},
		{Action: "delete", Keys: "leader d"},
		{Action: "clear_cards", Keys: "CmdOrCtrl+Shift+Option+Backspace"},
		{Action: "clear_cards", Keys: "leader Shift+D"},
		{Action: "pin", Keys: "CmdOrCtrl+Shift+F"},
		{Action: "pin", Keys: "leader p", Stay: true},
		{Action: "settings", Keys: "CmdOrCtrl+,"},
		{Action: "settings", Keys: "leader ,"},
	}
}

// BindingList is the shortcut list. It decodes the list form and the old tables.
type BindingList []Binding

// UnmarshalTOML accepts the list form and the old per-action tables.
func (b *BindingList) UnmarshalTOML(v any) error {
	switch val := v.(type) {
	case []map[string]any:
		rows, err := bindingsFromMaps(val)
		if err != nil {
			return err
		}
		*b = rows
		return nil
	case map[string]any:
		*b = migrateLegacyBindings(val)
		return nil
	default:
		return fmt.Errorf("bindings must be a list or the old action tables, got %T", v)
	}
}

func bindingsFromMaps(items []map[string]any) (BindingList, error) {
	out := make(BindingList, 0, len(items))
	for _, m := range items {
		action, _ := m["action"].(string)
		keys, _ := m["keys"].(string)
		stay, _ := m["stay"].(bool)
		out = append(out, Binding{Action: action, Keys: keys, Stay: stay})
	}
	return out, nil
}

// migrateLegacyBindings turns direct/leader/sticky fields into list rows.
// direct_double and leader_double have no list equivalent; the key remains
// and fires on a single press.
func migrateLegacyBindings(table map[string]any) BindingList {
	order := []string{"toggle", "next", "prev", "paste", "delete", "clear_cards", "pin", "settings"}
	var out BindingList
	for _, name := range order {
		raw, ok := table[name]
		if !ok {
			continue
		}
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if direct := strings.TrimSpace(asString(m["direct"])); direct != "" {
			out = append(out, Binding{Action: name, Keys: direct})
		}
		if leader := strings.TrimSpace(asString(m["leader"])); leader != "" {
			out = append(out, Binding{Action: name, Keys: "leader " + leader, Stay: asBool(m["sticky"])})
		}
	}
	return out
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func asBool(v any) bool {
	b, _ := v.(bool)
	return b
}

// finishLegacyLeader copies the old [leader].shortcut into the leader
// binding when the file still uses action tables, or when it has no
// binding list. A [[bindings]] list already names the leader key.
func (c *Config) finishLegacyLeader(data []byte) {
	array, table := bindingsKind(data)
	if array {
		c.Leader.Shortcut = ""
		return
	}
	shortcut := strings.TrimSpace(c.Leader.Shortcut)
	if shortcut == "" {
		shortcut = "CmdOrCtrl+Shift+Space"
	}
	if table || c.leaderRow() < 0 {
		rows := make(BindingList, 0, len(c.Bindings)+1)
		rows = append(rows, Binding{Action: "leader", Keys: shortcut})
		for _, row := range c.Bindings {
			if row.Action == "leader" {
				continue
			}
			rows = append(rows, row)
		}
		c.Bindings = rows
	} else if strings.TrimSpace(c.Leader.Shortcut) != "" {
		c.Bindings[c.leaderRow()].Keys = shortcut
	}
	c.Leader.Shortcut = ""
}

func (c Config) leaderRow() int {
	for i, row := range c.Bindings {
		if row.Action == "leader" {
			return i
		}
	}
	return -1
}

func bindingsKind(data []byte) (array, table bool) {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[[bindings]]") {
			array = true
		}
		if strings.HasPrefix(line, "[bindings.") || line == "[bindings]" {
			table = true
		}
	}
	return array, table
}

// KnownAction reports whether action is one snip can run.
func KnownAction(action string) bool {
	switch action {
	case "leader", "toggle", "next", "prev", "paste", "delete", "clear_cards", "pin", "settings":
		return true
	default:
		return false
	}
}

// Format renders a parsed shortcut in the form the file stores.
func (p ParsedBinding) Format() string {
	if !p.Leader {
		return p.Chord
	}
	chord := p.Chord
	if utf8.RuneCountInString(chord) == 1 {
		chord = strings.ToLower(chord)
	}
	return "leader " + chord
}

// CanonicalBindings rewrites each row to its stored spelling.
// Stay is kept only on leader combos.
func CanonicalBindings(rows []Binding) (BindingList, error) {
	out := make(BindingList, len(rows))
	for i, row := range rows {
		parsed, err := ParseBinding(row.Keys)
		if err != nil {
			return nil, err
		}
		row.Keys = parsed.Format()
		if !parsed.Leader {
			row.Stay = false
		}
		out[i] = row
	}
	return out, nil
}

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

// CheckBindings reports one issue per row. Empty and unknown keys win over
// a duplicate. A leader prefix on the leader action is its own error.
// Two rows conflict when they parse to the same global combo, or the same
// key after the leader key. A global combo does not conflict with a leader key.
func CheckBindings(rows []Binding) []*BindingIssue {
	out := make([]*BindingIssue, len(rows))
	seen := map[string]int{}
	for i, row := range rows {
		if !KnownAction(row.Action) {
			out[i] = &BindingIssue{Code: "action", Token: row.Action}
			continue
		}
		if row.Action == "leader" && HasLeaderPrefix(row.Keys) {
			out[i] = &BindingIssue{Code: "leader"}
			continue
		}
		parsed, err := ParseBinding(row.Keys)
		if err != nil {
			issue, ok := err.(*BindingIssue)
			if !ok {
				issue = &BindingIssue{Code: "unknown"}
			}
			out[i] = issue
			continue
		}
		id := parsed.Chord
		if parsed.Leader {
			id = "leader " + parsed.Chord
		}
		if prev, ok := seen[id]; ok {
			if out[prev] == nil {
				out[prev] = &BindingIssue{Code: "conflict"}
			}
			out[i] = &BindingIssue{Code: "conflict"}
			continue
		}
		seen[id] = i
	}
	return out
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
