package music

import (
	"testing"
)

func TestNoteNameToMIDI(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{"C0", 12},  {"C4", 60},  {"A4", 69},
		{"C5", 72},  {"E4", 64},  {"G4", 67},
		{"Db4", 61}, {"C#4", 61}, {"Bb3", 58},
		{"A#3", 58}, {"Eb4", 63}, {"D#4", 63},
		{"F4", 65},  {"B4", 71},  {"G#4", 68},
	}
	for _, tt := range tests {
		got, err := NoteNameToMIDI(tt.name)
		if err != nil {
			t.Errorf("NoteNameToMIDI(%q) error: %v", tt.name, err)
			continue
		}
		if got != tt.want {
			t.Errorf("NoteNameToMIDI(%q) = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestNoteNameToMIDI_Errors(t *testing.T) {
	_, err := NoteNameToMIDI("")
	if err == nil {
		t.Error("expected error for empty string")
	}
	_, err = NoteNameToMIDI("X4")
	if err == nil {
		t.Error("expected error for unknown note X")
	}
}

func TestMIDIToNoteName(t *testing.T) {
	tests := []struct {
		midi int
		want string
	}{
		{12, "C0"}, {60, "C4"}, {61, "C#4"},
		{69, "A4"}, {71, "B4"}, {72, "C5"},
		{48, "C3"}, {64, "E4"}, {67, "G4"},
	}
	for _, tt := range tests {
		got := MIDIToNoteName(tt.midi)
		if got != tt.want {
			t.Errorf("MIDIToNoteName(%d) = %q, want %q", tt.midi, got, tt.want)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	// Every MIDI note 0..127 should round-trip through note name.
	for midi := 0; midi <= 127; midi++ {
		name := MIDIToNoteName(midi)
		back, err := NoteNameToMIDI(name)
		if err != nil {
			t.Fatalf("round-trip failed at %d: name=%q, err=%v", midi, name, err)
		}
		if back != midi {
			t.Fatalf("round-trip mismatch at %d: got %d via %q", midi, back, name)
		}
	}
}

func TestGetScale(t *testing.T) {
	tests := []struct {
		root  string
		scale string
		want  []string
	}{
		{"C", "major", []string{"C", "D", "E", "F", "G", "A", "B"}},
		{"A", "minor", []string{"A", "B", "C", "D", "E", "F", "G"}},
		{"D", "natural_minor", []string{"D", "E", "F", "G", "A", "A#", "C"}},
		{"G", "major", []string{"G", "A", "B", "C", "D", "E", "F#"}},
	}
	for _, tt := range tests {
		got, err := GetScale(tt.root, tt.scale)
		if err != nil {
			t.Errorf("GetScale(%q, %q) error: %v", tt.root, tt.scale, err)
			continue
		}
		if len(got) != len(tt.want) {
			t.Errorf("GetScale(%q, %q) len=%d, want %d", tt.root, tt.scale, len(got), len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("GetScale(%q, %q)[%d] = %q, want %q", tt.root, tt.scale, i, got[i], tt.want[i])
			}
		}
	}
}

func TestGetScale_Errors(t *testing.T) {
	_, err := GetScale("C", "blues")
	if err == nil {
		t.Error("expected error for unsupported scale")
	}
	_, err = GetScale("X", "major")
	if err == nil {
		t.Error("expected error for unknown root")
	}
}

func TestParseChord(t *testing.T) {
	tests := []struct {
		chord string
		want  []string
	}{
		{"C", []string{"C", "E", "G"}},
		{"Cm", []string{"C", "D#", "G"}},
		{"D", []string{"D", "F#", "A"}},
		{"Dm", []string{"D", "F", "A"}},
		{"Bb", []string{"Bb", "D", "F"}},
		{"F#m", []string{"F#", "A", "C#"}},
	}
	for _, tt := range tests {
		got, err := ParseChord(tt.chord)
		if err != nil {
			t.Errorf("ParseChord(%q) error: %v", tt.chord, err)
			continue
		}
		if len(got) != len(tt.want) {
			t.Errorf("ParseChord(%q) len=%d, want %d", tt.chord, len(got), len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("ParseChord(%q)[%d] = %q, want %q", tt.chord, i, got[i], tt.want[i])
			}
		}
	}
}

func TestChordToMIDINotes(t *testing.T) {
	notes, err := ChordToMIDINotes("C", 4)
	if err != nil {
		t.Fatal(err)
	}
	want := []int{60, 64, 67}
	if len(notes) != len(want) {
		t.Fatalf("got %v, want %v", notes, want)
	}
	for i := range notes {
		if notes[i] != want[i] {
			t.Errorf("[%d] = %d, want %d", i, notes[i], want[i])
		}
	}

	// Minor chord: Am at octave 3 ->A3=57, C3=48, E3=52
	notes, _ = ChordToMIDINotes("Am", 3)
	want = []int{57, 48, 52}
	if len(notes) != len(want) {
		t.Fatalf("got %v, want %v", notes, want)
	}
}

func TestRootPitch(t *testing.T) {
	tests := []struct {
		chord string
		want  int
	}{
		{"C", 24},  {"C#", 25}, {"Db", 25},
		{"D", 26},  {"D#", 27}, {"Eb", 27},
		{"E", 28},  {"F", 29},  {"F#", 30},
		{"G", 31},  {"G#", 32}, {"A", 33},
		{"A#", 34}, {"B", 35},
		{"Cm", 24}, {"Dm", 26}, {"Am", 33},
	}
	for _, tt := range tests {
		got, err := RootPitch(tt.chord)
		if err != nil {
			t.Errorf("RootPitch(%q) error: %v", tt.chord, err)
			continue
		}
		if got != tt.want {
			t.Errorf("RootPitch(%q) = %d, want %d", tt.chord, got, tt.want)
		}
	}
}

func TestClampInt(t *testing.T) {
	if got := ClampInt(5, 0, 10); got != 5 {
		t.Errorf("clamp mid = %d, want 5", got)
	}
	if got := ClampInt(-1, 0, 10); got != 0 {
		t.Errorf("clamp low = %d, want 0", got)
	}
	if got := ClampInt(15, 0, 10); got != 10 {
		t.Errorf("clamp high = %d, want 10", got)
	}
}

func TestSemitoneConsistency(t *testing.T) {
	// Verify all enharmonic equivalents produce the same semitone.
	pairs := [][2]string{
		{"C#", "Db"}, {"D#", "Eb"}, {"F#", "Gb"},
		{"G#", "Ab"}, {"A#", "Bb"},
	}
	for _, p := range pairs {
		if noteToSemitone[p[0]] != noteToSemitone[p[1]] {
			t.Errorf("enharmonic mismatch: %s(%d) != %s(%d)",
				p[0], noteToSemitone[p[0]], p[1], noteToSemitone[p[1]])
		}
	}
}
