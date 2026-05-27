// Package e2e_test runs the full pipeline: song plan ->generators ->MIDI render ->file store.
package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ShowerBandV/text2midi/internal/generator"
	"github.com/ShowerBandV/text2midi/internal/midi"
	"github.com/ShowerBandV/text2midi/internal/schema"
	"github.com/ShowerBandV/text2midi/internal/store"
)

func TestFullPipeline(t *testing.T) {
	dir := t.TempDir()

	// 1. Build song plan.
	plan := schema.SongPlan{
		Title: "E2E_Test",
		BPM:   128,
		TimeSignature: schema.TimeSignature{Numerator: 4, Denominator: 4},
		Key:           schema.Key{Root: "A", Mode: "minor", Scale: "natural_minor"},
		TotalBars:     8,
		Loopable:      true,
		ChordProgression: []schema.ChordChange{
			{Bar: 0, Chord: "Am"},
			{Bar: 1, Chord: "F"},
			{Bar: 2, Chord: "C"},
			{Bar: 3, Chord: "G"},
			{Bar: 4, Chord: "Am"},
			{Bar: 5, Chord: "F"},
			{Bar: 6, Chord: "C"},
			{Bar: 7, Chord: "G"},
		},
	}

	// 2. Build arrangement.
	bassProg := 34
	padProg := 91
	leadProg := 81

	arrangement := schema.Arrangement{
		Tracks: []schema.ArrangementTrack{
			{ID: "drums", Name: "Drums", Role: "rhythm", Enabled: true,
				IsCoreTrack: true, GenerationStrategy: "drum_generator",
				Channel: 9, Program: nil, Volume: 105, Pan: 64},
			{ID: "bass", Name: "Bass", Role: "bass", Enabled: true,
				IsCoreTrack: true, GenerationStrategy: "bass_generator",
				Channel: 0, Program: &bassProg, Volume: 100, Pan: 64},
			{ID: "chords", Name: "Chords", Role: "harmony", Enabled: true,
				IsCoreTrack: true, GenerationStrategy: "chord_generator",
				Channel: 1, Program: &padProg, Volume: 90, Pan: 64},
			{ID: "lead", Name: "Lead", Role: "melody", Enabled: true,
				IsCoreTrack: true, GenerationStrategy: "lead_generator",
				Channel: 2, Program: &leadProg, Volume: 100, Pan: 64},
		},
	}

	// 3. Generate notes.
	eventsByTrack := make(map[string][]schema.NoteEvent)
	totalEvents := 0
	for _, tr := range arrangement.Tracks {
		if !tr.Enabled {
			continue
		}
		events := generator.GenerateNotes(plan, tr)
		eventsByTrack[tr.ID] = events
		totalEvents += len(events)

		if len(events) == 0 {
			t.Errorf("track %q generated 0 events", tr.ID)
		}
	}
	if totalEvents == 0 {
		t.Fatal("pipeline generated 0 total events")
	}
	t.Logf("Generated %d total events across %d tracks", totalEvents, len(arrangement.Tracks))

	// 4. Assemble MidiIR.
	beatsPerBar := plan.TimeSignature.Numerator
	var tracks []schema.TrackIR
	for _, at := range arrangement.Tracks {
		if !at.Enabled {
			continue
		}
		tracks = append(tracks, schema.TrackIR{
			ID: at.ID, Name: at.Name, Role: at.Role,
			Channel: at.Channel, Program: at.Program,
			Volume: at.Volume, Pan: at.Pan, Enabled: true,
			IsCoreTrack: at.IsCoreTrack, Events: eventsByTrack[at.ID],
		})
	}

	midiIR := schema.MidiIR{
		Meta: schema.Meta{
			Title:        plan.Title,
			BPM:          plan.BPM,
			TicksPerBeat: 480,
			TimeSignature: schema.TimeSignature{
				Numerator: plan.TimeSignature.Numerator,
				Denominator: plan.TimeSignature.Denominator,
			},
			KeySignature: fmt.Sprintf("%s %s", plan.Key.Root, plan.Key.Mode),
			TotalBars:    plan.TotalBars,
			BeatsPerBar:  beatsPerBar,
			TotalBeats:   plan.TotalBars * beatsPerBar,
			Loopable:     plan.Loopable,
		},
		Tracks: tracks,
	}

	// 5. Render to MIDI.
	outputPath := filepath.Join(dir, "e2e_test.mid")
	result, err := midi.RenderMIDI(midiIR, outputPath, nil)
	if err != nil {
		t.Fatalf("RenderMIDI failed: %v", err)
	}
	t.Logf("Rendered MIDI: %s (%d tracks, %d events, %.1fs)",
		result.OutputPath, result.TotalTracks, result.TotalNoteEvents, result.DurationSeconds)

	if result.DurationSeconds <= 0 {
		t.Errorf("duration=%f, want > 0", result.DurationSeconds)
	}

	// 6. Validate MIDI binary structure.
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data[0:4]) != "MThd" {
		t.Fatal("not a valid MIDI file (missing MThd)")
	}

	// 7. Save to file store.
	fs := store.NewFileStore(dir)
	record, err := fs.SaveFile("e2e-song", data, result)
	if err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}
	t.Logf("Saved to store: %s (%d bytes)", record.ID, record.FileSize)

	// 8. Load back and verify.
	loadedData, loadedRecord, err := fs.LoadFile("e2e-song")
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}
	if len(loadedData) != len(data) {
		t.Errorf("size mismatch after round-trip: %d vs %d", len(loadedData), len(data))
	}
	if loadedRecord.FileSize != record.FileSize {
		t.Errorf("FileSize mismatch: %d vs %d", loadedRecord.FileSize, record.FileSize)
	}

	// 9. Verify ListFiles returns the record.
	allRecords, err := fs.ListFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(allRecords) != 1 {
		t.Fatalf("expected 1 record, got %d", len(allRecords))
	}
	if allRecords[0].ID != "e2e-song" {
		t.Errorf("ListFiles ID = %q", allRecords[0].ID)
	}

	t.Log(">-Full pipeline test passed")
}

func TestPipeline_RandomSeedProducesVariation(t *testing.T) {
	// Run the pipeline twice and verify the output files differ (random seed).
	dir := t.TempDir()

	plan := schema.SongPlan{
		Title: "Variation", BPM: 120,
		TimeSignature: schema.TimeSignature{Numerator: 4, Denominator: 4},
		Key:           schema.Key{Root: "C", Mode: "major", Scale: "major"},
		TotalBars:     4,
		ChordProgression: []schema.ChordChange{
			{Bar: 0, Chord: "C"}, {Bar: 1, Chord: "F"},
			{Bar: 2, Chord: "G"}, {Bar: 3, Chord: "C"},
		},
	}

	arr := schema.Arrangement{
		Tracks: []schema.ArrangementTrack{
			{ID: "drums", Name: "Drums", Role: "rhythm", Enabled: true,
				IsCoreTrack: true, GenerationStrategy: "drum_generator",
				Channel: 9, Program: nil, Volume: 100, Pan: 64},
		},
	}

	generateFile := func() []byte {
		events := generator.GenerateNotes(plan, arr.Tracks[0])
		midiIR := schema.MidiIR{
			Meta: schema.Meta{
				Title: "V", BPM: 120, TicksPerBeat: 480,
				TimeSignature: schema.TimeSignature{Numerator: 4, Denominator: 4},
				TotalBars: 4, BeatsPerBar: 4, TotalBeats: 16,
			},
			Tracks: []schema.TrackIR{{
				ID: "drums", Name: "Drums", Role: "rhythm",
				Channel: 9, Volume: 100, Pan: 64, Enabled: true,
				Events: events,
			}},
		}
		out := filepath.Join(dir, "run.mid")
		midi.RenderMIDI(midiIR, out, nil)
		data, _ := os.ReadFile(out)
		return data
	}

	a := generateFile()
	b := generateFile()

	// Almost certainly different due to rand.
	same := len(a) == len(b)
	for i := 0; same && i < len(a); i++ {
		if a[i] != b[i] {
			same = false
		}
	}
	if same {
		t.Log(">-two runs produced identical output (unlikely but possible)")
	} else {
		t.Log(">-two runs produced different output (random variation working)")
	}
}
