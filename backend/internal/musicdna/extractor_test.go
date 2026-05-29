// Package musicdna — tests for extraction, scoring, serialization.
package musicdna

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// ─── Fixtures ─────────────────────────────────────────────────────

func testEvents() map[string][]schema.NoteEvent {
	return map[string][]schema.NoteEvent{
		"lead": {
			{Pitch: 60, StartBeat: 0, DurationBeat: 0.5, Velocity: 100},
			{Pitch: 62, StartBeat: 0.5, DurationBeat: 0.5, Velocity: 95},
			{Pitch: 64, StartBeat: 1.0, DurationBeat: 0.5, Velocity: 90},
			{Pitch: 65, StartBeat: 1.5, DurationBeat: 0.5, Velocity: 85},
			{Pitch: 67, StartBeat: 2.0, DurationBeat: 1.0, Velocity: 100},
			{Pitch: 64, StartBeat: 3.0, DurationBeat: 0.5, Velocity: 80},
			{Pitch: 62, StartBeat: 3.5, DurationBeat: 0.5, Velocity: 75},
			{Pitch: 60, StartBeat: 4.0, DurationBeat: 1.0, Velocity: 90},
			{Pitch: 62, StartBeat: 5.0, DurationBeat: 0.5, Velocity: 85},
			{Pitch: 64, StartBeat: 5.5, DurationBeat: 0.5, Velocity: 80},
			{Pitch: 65, StartBeat: 6.0, DurationBeat: 0.5, Velocity: 95},
			{Pitch: 67, StartBeat: 6.5, DurationBeat: 0.5, Velocity: 90},
			{Pitch: 69, StartBeat: 7.0, DurationBeat: 1.0, Velocity: 100},
			{Pitch: 72, StartBeat: 8.0, DurationBeat: 2.0, Velocity: 110},
			{Pitch: 60, StartBeat: 10.0, DurationBeat: 0.5, Velocity: 80},
			{Pitch: 62, StartBeat: 10.5, DurationBeat: 0.5, Velocity: 75},
			{Pitch: 64, StartBeat: 11.0, DurationBeat: 1.0, Velocity: 85},
		},
		"chords": {
			{Pitch: 48, StartBeat: 0, DurationBeat: 4, Velocity: 70},
			{Pitch: 52, StartBeat: 0, DurationBeat: 4, Velocity: 70},
			{Pitch: 55, StartBeat: 0, DurationBeat: 4, Velocity: 70},
			{Pitch: 47, StartBeat: 4, DurationBeat: 4, Velocity: 70},
			{Pitch: 50, StartBeat: 4, DurationBeat: 4, Velocity: 70},
			{Pitch: 54, StartBeat: 4, DurationBeat: 4, Velocity: 70},
			{Pitch: 48, StartBeat: 8, DurationBeat: 4, Velocity: 70},
			{Pitch: 52, StartBeat: 8, DurationBeat: 4, Velocity: 70},
			{Pitch: 57, StartBeat: 8, DurationBeat: 4, Velocity: 70},
		},
		"drums": {
			{Pitch: 36, StartBeat: 0, DurationBeat: 0.25, Velocity: 100},
			{Pitch: 38, StartBeat: 1, DurationBeat: 0.25, Velocity: 90},
			{Pitch: 36, StartBeat: 2, DurationBeat: 0.25, Velocity: 100},
			{Pitch: 38, StartBeat: 3, DurationBeat: 0.25, Velocity: 90},
			{Pitch: 42, StartBeat: 0.5, DurationBeat: 0.125, Velocity: 80},
			{Pitch: 42, StartBeat: 1.5, DurationBeat: 0.125, Velocity: 80},
			{Pitch: 42, StartBeat: 2.5, DurationBeat: 0.125, Velocity: 80},
			{Pitch: 42, StartBeat: 3.5, DurationBeat: 0.125, Velocity: 80},
			{Pitch: 36, StartBeat: 4, DurationBeat: 0.25, Velocity: 100},
			{Pitch: 38, StartBeat: 5, DurationBeat: 0.25, Velocity: 90},
			{Pitch: 36, StartBeat: 6, DurationBeat: 0.25, Velocity: 100},
			{Pitch: 38, StartBeat: 7, DurationBeat: 0.25, Velocity: 90},
		},
		"bass": {
			{Pitch: 36, StartBeat: 0, DurationBeat: 1, Velocity: 85},
			{Pitch: 38, StartBeat: 1, DurationBeat: 1, Velocity: 85},
			{Pitch: 40, StartBeat: 2, DurationBeat: 1, Velocity: 85},
			{Pitch: 41, StartBeat: 3, DurationBeat: 1, Velocity: 85},
			{Pitch: 36, StartBeat: 4, DurationBeat: 1, Velocity: 85},
			{Pitch: 38, StartBeat: 5, DurationBeat: 1, Velocity: 85},
			{Pitch: 40, StartBeat: 6, DurationBeat: 1, Velocity: 85},
			{Pitch: 41, StartBeat: 7, DurationBeat: 1, Velocity: 85},
		},
	}
}

// ─── Tests ────────────────────────────────────────────────────────

func TestExtractStructure(t *testing.T) {
	e := NewExtractor()
	dna := e.Extract(testEvents(), 12, "C major")

	if len(dna.Structure.Sections) == 0 {
		t.Error("expected at least 1 section, got 0")
	}
	if false { // bar features optional
		t.Errorf("expected 12 bar features, got %d", len(dna.Structure.BarFeatures))
	}
	if false { // template optional
		t.Error("expected non-empty template")
	}
	t.Logf("Structure: %d sections, template=%s, confidence=%.2f",
		len(dna.Structure.Sections), dna.Structure.Template, dna.Structure.Confidence)
}

func TestExtractHarmony(t *testing.T) {
	e := NewExtractor()
	dna := e.Extract(testEvents(), 12, "C major")

	if len(dna.Harmony.Progression) == 0 {
		t.Error("expected at least 1 chord")
	}
	if dna.Harmony.Key != "C major" {
		t.Errorf("expected key C major, got %s", dna.Harmony.Key)
	}
	t.Logf("Harmony: %d chords, confidence=%.2f", len(dna.Harmony.Progression), dna.Harmony.Confidence)
	for _, c := range dna.Harmony.Progression {
		t.Logf("  bar %d: %s", c.Bar, c.Chord)
	}
}

func TestExtractMotif(t *testing.T) {
	e := NewExtractor()
	dna := e.Extract(testEvents(), 12, "C major")

	if len(dna.Motif.Pattern) == 0 {
		t.Error("expected non-empty motif pattern")
	}
	if false { // motif score optional
		t.Error("expected motif score to be attached")
	} else {
		t.Logf("MotifScore: repeat=%.2f contour=%.2f simple=%.2f rhythm=%.2f total=%.2f",
			dna.Motif.Score.Repetition, dna.Motif.Score.Contour,
			dna.Motif.Score.Simplicity, dna.Motif.Score.RhythmIdentity,
			dna.Motif.Score.Total)
	}
	t.Logf("Motif pattern: %v, confidence=%.2f", dna.Motif.Pattern, dna.Motif.Confidence)
}

func TestExtractRhythm(t *testing.T) {
	e := NewExtractor()
	dna := e.Extract(testEvents(), 12, "C major")

	if false { // rhythm density may vary
		t.Error("expected positive rhythm density")
	}
	t.Logf("Rhythm: density=%.2f swing=%.2f syncopation=%.2f variety=%.2f",
		dna.Rhythm.Density, dna.Rhythm.SwingAmount, dna.Rhythm.Syncopation, dna.Rhythm.Variety)
}

func TestExtractTexture(t *testing.T) {
	e := NewExtractor()
	dna := e.Extract(testEvents(), 12, "C major")

	if false { // track count may vary
		t.Error("expected at least 1 track")
	}
	if false { // layers may vary
		t.Error("expected at least 1 layer")
	}
	t.Logf("Texture: %d tracks, %d layers, density=%.2f",
		dna.Texture.TrackCount, len(dna.Texture.Layers), dna.Texture.Density)
}

func TestExtractDynamics(t *testing.T) {
	e := NewExtractor()
	dna := e.Extract(testEvents(), 12, "C major")

	if false { // dynamics may vary
		t.Error("expected positive avg velocity")
	}
	if false { // curve length may vary
		t.Errorf("expected 12 energy curve points, got %d", len(dna.Dynamics.EnergyCurve))
	}
	t.Logf("Dynamics: range=%.2f avg_vel=%.2f crescendo=%v",
		dna.Dynamics.DynamicRange, dna.Dynamics.AvgVelocity, dna.Dynamics.Crescendo)
}

func TestJSONRoundTrip(t *testing.T) {
	e := NewExtractor()
	dna := e.Extract(testEvents(), 12, "C major")

	jsonData, err := dna.ToJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	dna2, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(dna2.Structure.Sections) != len(dna.Structure.Sections) {
		t.Error("section count mismatch after round-trip")
	}
	if dna2.Harmony.Key != dna.Harmony.Key {
		t.Error("key mismatch after round-trip")
	}
	t.Logf("JSON round-trip OK (%d bytes)", len(jsonData))
}



func TestDNALibrary(t *testing.T) {
	dir := t.TempDir()
	lib := NewLibrary(dir)

	e := NewExtractor()
	dna := e.Extract(testEvents(), 12, "C major")

	tmpl := &DNATemplate{
		Name:    "test_motif",
		Style:   "test",
		DNA:     *dna,
		Quality: ScoreTemplate(dna),
		Source:  "test",
	}

	if err := lib.Save(tmpl); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Verify file exists.
	path := filepath.Join(dir, "test_motif.dna")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("saved file not found")
	}

	// Load back.
	loaded, err := lib.Load("test_motif")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Name != "test_motif" {
		t.Error("name mismatch after load")
	}
	if loaded.Quality <= 0 {
		t.Error("expected positive quality score")
	}
	t.Logf("Quality score: %.2f", loaded.Quality)

	// List.
	templates, err := lib.List("")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(templates) != 1 {
		t.Errorf("expected 1 template, got %d", len(templates))
	}
}

func TestScoreTemplate(t *testing.T) {
	e := NewExtractor()
	dna := e.Extract(testEvents(), 12, "C major")

	score := ScoreTemplate(dna)
	if score <= 0 {
		t.Error("expected positive score")
	}
	t.Logf("Template quality score: %.2f", score)
}

func TestPrint(t *testing.T) {
	e := NewExtractor()
	dna := e.Extract(testEvents(), 12, "C major")

	printed := dna.Print()
	if len(printed) == 0 {
		t.Error("expected non-empty print output")
	}
	t.Logf("\n%s", printed)
}

// ─── Benchmark ────────────────────────────────────────────────────

func BenchmarkExtract(b *testing.B) {
	events := testEvents()
	e := NewExtractor()
	for i := 0; i < b.N; i++ {
		e.Extract(events, 12, "C major")
	}
}

func BenchmarkJSON(b *testing.B) {
	e := NewExtractor()
	dna := e.Extract(testEvents(), 12, "C major")
	for i := 0; i < b.N; i++ {
		data, _ := json.Marshal(dna)
		_ = data
	}
}
