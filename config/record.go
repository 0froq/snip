package config

import "time"

// LeaderWait is how long a recording waits for the key after the leader chord.
const LeaderWait = 2 * time.Second

// LeaderWaitText is the field text while that second chord is still to come.
const LeaderWaitText = "leader …"

// RecPhase is the shortcut recorder's phase.
type RecPhase int

const (
	RecIdle RecPhase = iota
	RecCapture
	RecWait
)

// RecMods is the modifier set held when a non-modifier key goes down.
type RecMods struct {
	Cmd   bool
	Ctrl  bool
	Alt   bool
	Shift bool
}

func (m RecMods) set() modSet {
	var mods modSet
	if m.Cmd {
		mods |= modCmd
	}
	if m.Ctrl {
		mods |= modCtrl
	}
	if m.Alt {
		mods |= modAlt
	}
	if m.Shift {
		mods |= modShift
	}
	return mods
}

// Recorder captures one shortcut chord, or a leader chord and its follow-up.
// The text field is the value in Text. Esc restores Previous. Backspace clears Text.
type Recorder struct {
	On        bool
	Row       int
	Action    string
	Previous  string
	Phase     RecPhase
	Text      string
	Issue     string
	WaitUntil time.Time

	down     bool
	downMods modSet
	downKey  string
}

// Start begins recording row. previous is restored if the recording is cancelled.
func (r *Recorder) Start(row int, action, previous string) {
	*r = Recorder{
		On:       true,
		Row:      row,
		Action:   action,
		Previous: previous,
		Phase:    RecCapture,
		Text:     previous,
	}
}

// Esc cancels the recording and restores the previous field text.
func (r *Recorder) Esc() {
	if !r.On {
		return
	}
	prev := r.Previous
	row := r.Row
	*r = Recorder{Row: row, Text: prev, Previous: prev}
}

// Backspace clears the field and keeps recording the next chord.
func (r *Recorder) Backspace() {
	if !r.On {
		return
	}
	r.Text = ""
	r.Issue = ""
	r.Phase = RecCapture
	r.down = false
	r.WaitUntil = time.Time{}
}

// Down remembers a non-modifier key and the modifiers held with it.
func (r *Recorder) Down(mods RecMods, key string) {
	if !r.On || key == "" {
		return
	}
	r.down = true
	r.downMods = mods.set()
	r.downKey = key
}

// Up completes the chord when key is the one Down remembered.
// A chord equal to a configured leader chord, on any row but the leader row,
// waits for a follow-up instead of finishing.
// finished is true when the field is ready to validate.
func (r *Recorder) Up(key string, now time.Time, leaderChords []string) (finished bool) {
	if !r.On || !r.down || key == "" || key != r.downKey {
		return false
	}
	r.down = false
	chord, err := canonicalChord(r.downMods.String() + r.downKey)
	if err != nil {
		return false
	}
	if r.Phase == RecWait {
		r.Text = (ParsedBinding{Leader: true, Chord: chord}).Format()
		r.Issue = ""
		r.On = false
		r.Phase = RecIdle
		return true
	}
	if r.Action != "leader" && chordListed(chord, leaderChords) {
		r.Text = LeaderWaitText
		r.Phase = RecWait
		r.WaitUntil = now.Add(LeaderWait)
		r.Issue = ""
		return false
	}
	r.Text = (ParsedBinding{Chord: chord}).Format()
	r.Issue = ""
	r.On = false
	r.Phase = RecIdle
	return true
}

// Poll ends a leader wait that has run out of time.
// The field stays "leader" and Issue is leader_follow.
func (r *Recorder) Poll(now time.Time) (finished bool) {
	if !r.On || r.Phase != RecWait {
		return false
	}
	if now.Before(r.WaitUntil) {
		return false
	}
	r.Text = "leader"
	r.Issue = "leader_follow"
	r.On = false
	r.Phase = RecIdle
	r.down = false
	return true
}

func chordListed(chord string, chords []string) bool {
	for _, c := range chords {
		if c == chord {
			return true
		}
	}
	return false
}
