// Package schema defines the core data structures for the music generation pipeline.
// These are the Go equivalent of the Python dataclasses in core/schema.py.
package schema

// TimeSignature represents a musical time signature like 4/4.
type TimeSignature struct {
	Numerator   int `json:"numerator"`
	Denominator int `json:"denominator"`
}

// Key represents a musical key (root + mode).
type Key struct {
	Root  string `json:"root"`
	Mode  string `json:"mode"`
	Scale string `json:"scale"`
}

// ChordChange represents one chord in a progression at a specific bar.
type ChordChange struct {
	Bar   int    `json:"bar"`
	Chord string `json:"chord"`
}

// PitchBendEvent represents a MIDI pitch bend at a bar position.
// Value range: 0-16383, center=8192 (no bend).
type PitchBendEvent struct {
	Bar   float64 `json:"bar"`
	Value int     `json:"value"` // 0-16383, 8192=center
}

// CCEvent represents a MIDI control change to insert in a track.
type CCEvent struct {
	Bar      float64 `json:"bar"`      // bar position (float for fractional bars)
	Controller int    `json:"controller"` // CC number: 11=expression, 64=sustain, 1=modulation
	Value    int    `json:"value"`     // 0-127
}

// NoteEvent represents a single note in a track.
type NoteEvent struct {
	Type        string  `json:"type"`  // "note"
	Pitch       int     `json:"pitch"` // MIDI pitch (0-127)
	StartBeat   float64 `json:"start_beat"`
	DurationBeat float64 `json:"duration_beat"`
	Velocity    int     `json:"velocity"` // 1-127
	NoteName    string  `json:"note_name,omitempty"`
	DrumName    string  `json:"drum_name,omitempty"`
}

// TrackIR is the intermediate representation for a single track.
type TrackIR struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Role        string      `json:"role"`
	Channel     int         `json:"channel"`
	Program     *int        `json:"program,omitempty"`
	Volume      int         `json:"volume"`
	Pan         int         `json:"pan"`
	Enabled     bool        `json:"enabled"`
	IsCoreTrack bool        `json:"is_core_track"`
	Events         []NoteEvent       `json:"events"`
	CCEvents       []CCEvent         `json:"cc_events,omitempty"`       // control change events
	PitchBendEvents []PitchBendEvent `json:"pitch_bend_events,omitempty"` // pitch bend
}

// Meta holds metadata for the entire composition.
type Meta struct {
	Title          string        `json:"title"`
	BPM            int           `json:"bpm"`
	TicksPerBeat   int           `json:"ticks_per_beat"`
	TimeSignature  TimeSignature `json:"time_signature"`
	KeySignature   string        `json:"key_signature"`
	TotalBars      int           `json:"total_bars"`
	BeatsPerBar    int           `json:"beats_per_bar"`
	TotalBeats     int           `json:"total_beats"`
	Loopable       bool          `json:"loopable"`
}

// MidiIR is the complete intermediate representation of a composition.
type MidiIR struct {
	Meta   Meta      `json:"meta"`
	Tracks []TrackIR `json:"tracks"`
}

// SongSection describes one section (intro/verse/chorus/...) in the song structure.
type SongSection struct {
	Name     string  `json:"name"`     // "intro", "verse", "chorus", "bridge", "outro"
	StartBar int     `json:"start_bar"`
	Bars     int     `json:"bars"`
	Energy   float64 `json:"energy"`   // 0-1 target energy for this section
	Density  float64 `json:"density"`  // 0-1 note density hint
	Register string  `json:"register"` // "low", "mid", "high" — target pitch register
}

// SongPlan represents the high-level song structure.
// This is what the LLM song_planner would produce (or a hardcoded test version).
type SongPlan struct {
	Title            string         `json:"title"`
	BPM              int            `json:"bpm"`
	TimeSignature    TimeSignature  `json:"time_signature"`
	Key              Key            `json:"key"`
	TotalBars        int            `json:"total_bars"`
	Loopable         bool           `json:"loopable"`
	ChordProgression []ChordChange  `json:"chord_progression"`
	Sections         []SongSection  `json:"sections,omitempty"`
	EstimatedDuration float64       `json:"estimated_duration_seconds"`
	FeatureVector    FeatureVector  `json:"feature_vector,omitempty"`
}

// ArrangementTrack represents one track in the arrangement.
type ArrangementTrack struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Role               string `json:"role"`
	Enabled            bool   `json:"enabled"`
	IsCoreTrack        bool   `json:"is_core_track"`
	GenerationStrategy string `json:"generation_strategy"`
	Channel            int    `json:"channel"`
	Program            *int   `json:"program,omitempty"`
	Volume             int    `json:"volume"`
	Pan                int    `json:"pan"`
}

// FeatureVector encodes the musical character of a composition across 7 dimensions.
// Each dimension is 0.0--.0, making it LLM-friendly and generator-actionable.
// Style defaults provide a baseline; the LLM Intent Parser adjusts per user prompt.
type FeatureVector struct {
	Darkness           float64 `json:"darkness"`
	Energy             float64 `json:"energy"`
	Acousticness       float64 `json:"acousticness"`
	Density            float64 `json:"density"`
	RhythmicComplexity float64 `json:"rhythmic_complexity"`
	Tension            float64 `json:"tension"`
	LoFi               float64 `json:"lo_fi"`
}
// Arrangement holds all tracks for a composition.
type Arrangement struct {
	Tracks []ArrangementTrack `json:"tracks"`
}
