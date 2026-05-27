package agent

import (
	"encoding/json"
	"testing"
)

func TestMapToSongPlan_Basic(t *testing.T) {
	input := `{
		"title": "Test Track",
		"bpm": 128,
		"time_signature": {"numerator": 4, "denominator": 4},
		"key": {"root": "A", "mode": "minor", "scale": "natural_minor"},
		"total_bars": 16,
		"loopable": true,
		"estimated_duration_seconds": 30,
		"chord_progression": [
			{"bar": 0, "chord": "Am"},
			{"bar": 1, "chord": "F"},
			{"bar": 2, "chord": "C"},
			{"bar": 3, "chord": "G"}
		]
	}`

	var m map[string]any
	if err := json.Unmarshal([]byte(input), &m); err != nil {
		t.Fatal(err)
	}

	plan, err := mapToSongPlan(m)
	if err != nil {
		t.Fatalf("mapToSongPlan failed: %v", err)
	}

	if plan.Title != "Test Track" {
		t.Errorf("Title = %q", plan.Title)
	}
	if plan.BPM != 128 {
		t.Errorf("BPM = %d", plan.BPM)
	}
	if plan.Key.Root != "A" {
		t.Errorf("Key.Root = %q", plan.Key.Root)
	}
	if plan.Key.Mode != "minor" {
		t.Errorf("Key.Mode = %q", plan.Key.Mode)
	}
	if plan.TotalBars != 16 {
		t.Errorf("TotalBars = %d", plan.TotalBars)
	}
	if len(plan.ChordProgression) != 4 {
		t.Errorf("ChordProgression len = %d", len(plan.ChordProgression))
	}
	if plan.ChordProgression[0].Chord != "Am" {
		t.Errorf("first chord = %q", plan.ChordProgression[0].Chord)
	}
}

func TestMapToSongPlan_Defaults(t *testing.T) {
	// Minimal input --should fill defaults.
	input := `{
		"title": "Minimal",
		"bpm": 120,
		"key": {"root": "C", "mode": "major"},
		"total_bars": 8,
		"chord_progression": [{"bar": 0, "chord": "C"}, {"bar": 1, "chord": "G"}]
	}`

	var m map[string]any
	json.Unmarshal([]byte(input), &m)
	plan, err := mapToSongPlan(m)
	if err != nil {
		t.Fatalf("mapToSongPlan failed: %v", err)
	}

	if plan.Key.Scale != "major" {
		t.Errorf("Key.Scale = %q, want 'major' (inferred from mode)", plan.Key.Scale)
	}
	if plan.TimeSignature.Numerator != 4 {
		t.Errorf("Time sig = %d/4 (default)", plan.TimeSignature.Numerator)
	}
	if plan.EstimatedDuration <= 0 {
		t.Error("EstimatedDuration should be > 0")
	}
}

func TestMapToSongPlan_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", `{}`},
		{"no bpm", `{"title":"X","key":{"root":"C"},"total_bars":8,"chord_progression":[{"bar":0,"chord":"C"}]}`},
		{"no key", `{"title":"X","bpm":120,"total_bars":8,"chord_progression":[{"bar":0,"chord":"C"}]}`},
		{"no chords", `{"title":"X","bpm":120,"key":{"root":"C"},"total_bars":8,"chord_progression":[]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m map[string]any
			json.Unmarshal([]byte(tt.input), &m)
			_, err := mapToSongPlan(m)
			if err == nil {
				t.Error("expected error for invalid input")
			}
		})
	}
}

func TestMapToArrangementTrack_Basic(t *testing.T) {
	input := `{
		"name": "Lead",
		"role": "melody",
		"enabled": true,
		"is_core_track": true,
		"generation_strategy": "lead_generator",
		"midi": {"channel": 0, "program": 80},
		"mix": {"volume": 100, "pan": 64},
		"style": {"pattern_type": "lead"},
		"sections": {}
	}`

	var m map[string]any
	json.Unmarshal([]byte(input), &m)

	track, err := mapToArrangementTrack("lead", m)
	if err != nil {
		t.Fatalf("mapToArrangementTrack failed: %v", err)
	}

	if track.ID != "lead" {
		t.Errorf("ID = %q", track.ID)
	}
	if track.Channel != 0 {
		t.Errorf("Channel = %d", track.Channel)
	}
	if track.Program == nil || *track.Program != 80 {
		t.Errorf("Program = %v", track.Program)
	}
	if track.Volume != 100 {
		t.Errorf("Volume = %d", track.Volume)
	}
}

func TestMapToArrangementTrack_Drums(t *testing.T) {
	input := `{
		"name": "Drums",
		"role": "rhythm",
		"enabled": true,
		"is_core_track": true,
		"generation_strategy": "drum_generator",
		"midi": {"channel": 9, "program": null},
		"mix": {"volume": 105, "pan": 64},
		"style": {},
		"sections": {}
	}`

	var m map[string]any
	json.Unmarshal([]byte(input), &m)

	track, err := mapToArrangementTrack("drums", m)
	if err != nil {
		t.Fatal(err)
	}
	if track.Channel != 9 {
		t.Errorf("drum channel = %d", track.Channel)
	}
	if track.Program != nil {
		t.Errorf("drum program should be nil, got %v", track.Program)
	}
}

func TestMapToArrangementTrack_InvalidChannel(t *testing.T) {
	input := `{
		"name": "Bad",
		"role": "test",
		"midi": {"channel": 99, "program": 0},
		"mix": {"volume": 100, "pan": 64}
	}`
	var m map[string]any
	json.Unmarshal([]byte(input), &m)
	_, err := mapToArrangementTrack("bad", m)
	if err == nil {
		t.Error("expected error for invalid channel")
	}
}

func TestMapToArrangementTrack_MissingMidi(t *testing.T) {
	input := `{"name": "Bad", "role": "test", "mix": {"volume": 100, "pan": 64}}`
	var m map[string]any
	json.Unmarshal([]byte(input), &m)
	_, err := mapToArrangementTrack("bad", m)
	if err == nil {
		t.Error("expected error for missing midi")
	}
}

func TestGetString(t *testing.T) {
	m := map[string]any{"a": "hello", "b": 42, "c": nil}
	if got := getString(m, "a"); got != "hello" {
		t.Errorf("got %q", got)
	}
	if got := getString(m, "b"); got != "" {
		t.Errorf("expected empty for non-string, got %q", got)
	}
	if got := getString(m, "c"); got != "" {
		t.Errorf("expected empty for nil, got %q", got)
	}
	if got := getString(m, "nonexistent"); got != "" {
		t.Errorf("expected empty for missing, got %q", got)
	}
}

func TestGetInt(t *testing.T) {
	m := map[string]any{"a": 42.0, "b": 100}
	if got := getInt(m, "a"); got != 42 {
		t.Errorf("got %d", got)
	}
	if got := getInt(m, "b"); got != 100 {
		t.Errorf("got %d", got)
	}
	if got := getInt(m, "missing"); got != 0 {
		t.Errorf("got %d", got)
	}
}

func TestGetBool(t *testing.T) {
	m := map[string]any{"a": true, "b": false, "c": 1}
	if !getBool(m, "a") {
		t.Error("a should be true")
	}
	if getBool(m, "b") {
		t.Error("b should be false")
	}
	if getBool(m, "c") {
		t.Error("c should be false (non-bool)")
	}
}

func TestToFloat64(t *testing.T) {
	if got := toFloat64(42.5); got != 42.5 {
		t.Errorf("got %f", got)
	}
	if got := toFloat64(42); got != 42.0 {
		t.Errorf("got %f", got)
	}
	if got := toFloat64("bad"); got != 0.0 {
		t.Errorf("got %f", got)
	}
}
