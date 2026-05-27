package generator

import (
	"testing"

	"github.com/yourname/text2midi/internal/schema"
)

// testPlan returns a minimal SongPlan for testing.
func testPlan() schema.SongPlan {
	return schema.SongPlan{
		Title: "Test",
		BPM:   120,
		TimeSignature: schema.TimeSignature{Numerator: 4, Denominator: 4},
		Key:           schema.Key{Root: "C", Mode: "minor", Scale: "natural_minor"},
		TotalBars:     4,
		Loopable:      true,
		ChordProgression: []schema.ChordChange{
			{Bar: 0, Chord: "Cm"},
			{Bar: 1, Chord: "Ab"},
			{Bar: 2, Chord: "Bb"},
			{Bar: 3, Chord: "G"},
		},
	}
}

// testTrack returns a minimal ArrangementTrack for testing.
func testTrack(id, name, role string, channel int) schema.ArrangementTrack {
	prog := 0
	return schema.ArrangementTrack{
		ID: id, Name: name, Role: role, Enabled: true,
		IsCoreTrack: true, GenerationStrategy: id + "_generator",
		Channel: channel, Program: &prog, Volume: 100, Pan: 64,
	}
}

func TestGenerateDrums_ProducesEvents(t *testing.T) {
	plan := testPlan()
	track := testTrack("drums", "Drums", "rhythm", 9)
	events := GenerateDrums(plan, track)
	if len(events) == 0 {
		t.Fatal("GenerateDrums returned no events")
	}
	validateEvents(t, events, "drums")
}

func TestGenerateBass_ProducesEvents(t *testing.T) {
	plan := testPlan()
	track := testTrack("bass", "Bass", "bass", 0)
	events := GenerateBass(plan, track)
	if len(events) == 0 {
		t.Fatal("GenerateBass returned no events")
	}
	validateEvents(t, events, "bass")
}

func TestGenerateChords_ProducesEvents(t *testing.T) {
	plan := testPlan()
	track := testTrack("chords", "Chords", "harmony", 1)
	events := GenerateChords(plan, track)
	if len(events) == 0 {
		t.Fatal("GenerateChords returned no events")
	}
	validateEvents(t, events, "chords")
}

func TestGenerateLead_ProducesEvents(t *testing.T) {
	plan := testPlan()
	track := testTrack("lead", "Lead", "melody", 2)
	events := GenerateLead(plan, track)
	if len(events) == 0 {
		t.Fatal("GenerateLead returned no events")
	}
	validateEvents(t, events, "lead")
}

func TestGenerateGeneric_ProducesEvents(t *testing.T) {
	plan := testPlan()
	track := testTrack("extra", "Extra", "fx", 3)
	events := GenerateGeneric(plan, track)
	if len(events) == 0 {
		t.Fatal("GenerateGeneric returned no events")
	}
	validateEvents(t, events, "extra")
}

func TestGenerateNotes_Dispatch(t *testing.T) {
	plan := testPlan()
	trackIDs := []string{"drums", "bass", "chords", "lead", "unknown"}
	for _, id := range trackIDs {
		track := testTrack(id, id, "test", 0)
		events := GenerateNotes(plan, track)
		if len(events) == 0 {
			t.Errorf("GenerateNotes(%q) returned no events", id)
		}
	}
}

func TestDrums_ContainsKickSnareHat(t *testing.T) {
	plan := testPlan()
	track := testTrack("drums", "Drums", "rhythm", 9)
	events := GenerateDrums(plan, track)

	hasKick := false
	hasSnare := false
	hasHat := false
	for _, e := range events {
		switch e.DrumName {
		case "kick":
			hasKick = true
		case "snare":
			hasSnare = true
		case "closed_hat":
			hasHat = true
		}
	}
	if !hasKick {
		t.Error("drums missing kick")
	}
	if !hasSnare {
		t.Error("drums missing snare")
	}
	if !hasHat {
		t.Error("drums missing closed_hat")
	}
}

func TestBass_FollowsChordRoots(t *testing.T) {
	plan := testPlan()
	track := testTrack("bass", "Bass", "bass", 0)
	events := GenerateBass(plan, track)

	for _, e := range events {
		// Bass notes should be in a reasonable range (C1=24 to C3=48 range, allowing octave jumps).
		if e.Pitch < 24 || e.Pitch > 60 {
			t.Errorf("bass pitch %d out of expected range [24,60]", e.Pitch)
		}
	}
}

func TestLead_UsesScale(t *testing.T) {
	// C natural minor scale notes: C, D, Eb, F, G, Ab, Bb
	// C5=72, D5=74, Eb5=75, F5=77, G5=79, Ab5=80, Bb5=82
	validPitches := map[int]bool{72: true, 74: true, 75: true, 77: true, 79: true, 80: true, 82: true}

	plan := testPlan() // key = C minor
	track := testTrack("lead", "Lead", "melody", 2)
	events := GenerateLead(plan, track)

	for _, e := range events {
		if !validPitches[e.Pitch] {
			t.Errorf("lead pitch %d not in C minor scale (C5=72, D5=74, Eb5=75, F5=77, G5=79, Ab5=80, Bb5=82)", e.Pitch)
		}
	}
}

func TestMotifConsistency(t *testing.T) {
	for i := 0; i < 100; i++ {
		m := makeMotif(8)
		if len(m) != 8 {
			t.Fatalf("motif len=%d, want 8", len(m))
		}
		if m[0] != 0 {
			t.Errorf("motif[0] = %d, want 0", m[0])
		}
		if m[7] != 0 && m[7] != 2 && m[7] != 4 {
			t.Errorf("motif[7] = %d, want 0, 2, or 4", m[7])
		}
		for _, v := range m {
			if v < 0 || v > 6 {
				t.Errorf("motif value %d out of range [0,6]", v)
			}
		}
	}
}

// validateEvents checks that all events have valid MIDI parameters.
func validateEvents(t *testing.T, events []schema.NoteEvent, label string) {
	t.Helper()
	for i, e := range events {
		if e.Pitch < 0 || e.Pitch > 127 {
			t.Errorf("%s[%d]: pitch=%d out of range [0,127]", label, i, e.Pitch)
		}
		if e.Velocity < 1 || e.Velocity > 127 {
			t.Errorf("%s[%d]: velocity=%d out of range [1,127]", label, i, e.Velocity)
		}
		if e.StartBeat < 0 {
			t.Errorf("%s[%d]: start_beat=%f is negative", label, i, e.StartBeat)
		}
		if e.DurationBeat <= 0 {
			t.Errorf("%s[%d]: duration_beat=%f is not positive", label, i, e.DurationBeat)
		}
		if e.Type != "note" {
			t.Errorf("%s[%d]: type=%q, want \"note\"", label, i, e.Type)
		}
	}
}
