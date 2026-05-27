package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ShowerBandV/text2midi/internal/midi"
	"github.com/ShowerBandV/text2midi/internal/schema"
)

func TestSaveAndLoadFile(t *testing.T) {
	dir := t.TempDir()
	fs := NewFileStore(dir)

	data := []byte{0x4D, 0x54, 0x68, 0x64, 0x00, 0x00, 0x00, 0x06} // fake MIDI header
	meta := &midi.RenderResult{
		OutputPath:      "test.mid",
		TicksPerBeat:    480,
		TotalTracks:     1,
		TotalNoteEvents: 5,
		DurationSeconds: 10.0,
	}

	record, err := fs.SaveFile("test-song", data, meta)
	if err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}

	if record.ID != "test-song" {
		t.Errorf("record.ID = %q, want %q", record.ID, "test-song")
	}
	if record.FileSize != int64(len(data)) {
		t.Errorf("record.FileSize = %d, want %d", record.FileSize, len(data))
	}
	if record.RenderMeta == nil {
		t.Fatal("record.RenderMeta is nil")
	}
	if record.RenderMeta.DurationSeconds != 10.0 {
		t.Errorf("DurationSeconds = %f, want 10.0", record.RenderMeta.DurationSeconds)
	}

	// Load back
	loadedData, loadedRecord, err := fs.LoadFile("test-song")
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	if len(loadedData) != len(data) {
		t.Errorf("loaded data len = %d, want %d", len(loadedData), len(data))
	}
	for i := range data {
		if loadedData[i] != data[i] {
			t.Errorf("loadedData[%d] = %02x, want %02x", i, loadedData[i], data[i])
		}
	}

	if loadedRecord.ID != record.ID {
		t.Errorf("loaded ID = %q, want %q", loadedRecord.ID, record.ID)
	}
}

func TestLoadMeta(t *testing.T) {
	dir := t.TempDir()
	fs := NewFileStore(dir)

	data := []byte{0x4D, 0x54, 0x68, 0x64}
	_, err := fs.SaveFile("meta-test", data, nil)
	if err != nil {
		t.Fatal(err)
	}

	record, err := fs.LoadMeta("meta-test")
	if err != nil {
		t.Fatalf("LoadMeta failed: %v", err)
	}
	if record.ID != "meta-test" {
		t.Errorf("ID = %q", record.ID)
	}
}

func TestLoadNonexistent(t *testing.T) {
	dir := t.TempDir()
	fs := NewFileStore(dir)

	_, _, err := fs.LoadFile("nope")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}

	_, err = fs.LoadMeta("nope")
	if err == nil {
		t.Fatal("expected error for nonexistent meta")
	}
}

func TestListFiles(t *testing.T) {
	dir := t.TempDir()
	fs := NewFileStore(dir)

	// No files initially.
	records, err := fs.ListFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 files, got %d", len(records))
	}

	// Save 2 files.
	fs.SaveFile("song1", []byte{0x01}, nil)
	fs.SaveFile("song2", []byte{0x02}, nil)

	records, err = fs.ListFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 files, got %d", len(records))
	}

	ids := map[string]bool{}
	for _, r := range records {
		ids[r.ID] = true
	}
	if !ids["song1"] || !ids["song2"] {
		t.Errorf("ListFiles missing expected ids: got %v", ids)
	}
}

func TestDeleteFile(t *testing.T) {
	dir := t.TempDir()
	fs := NewFileStore(dir)

	fs.SaveFile("delete-me", []byte{0x01}, nil)

	// Verify it exists.
	_, _, err := fs.LoadFile("delete-me")
	if err != nil {
		t.Fatal(err)
	}

	// Delete it.
	if err := fs.DeleteFile("delete-me"); err != nil {
		t.Fatalf("DeleteFile failed: %v", err)
	}

	// Verify it's gone.
	_, _, err = fs.LoadFile("delete-me")
	if err == nil {
		t.Fatal("expected error after delete")
	}

	// List should be empty.
	records, _ := fs.ListFiles()
	if len(records) != 0 {
		t.Errorf("expected 0 files after delete, got %d", len(records))
	}
}

func TestRoundTrip_WithMIDIRender(t *testing.T) {
	// This test combines rendering a real MIDI file with store round-trip.
	dir := t.TempDir()
	fs := NewFileStore(dir)

	p := 0
	mid := schema.MidiIR{
		Meta: schema.Meta{
			Title: "RT", BPM: 120, TicksPerBeat: 480,
			TimeSignature: schema.TimeSignature{Numerator: 4, Denominator: 4},
			TotalBars: 1, BeatsPerBar: 4, TotalBeats: 4, Loopable: true,
		},
		Tracks: []schema.TrackIR{
			{
				ID: "t1", Name: "Track1", Role: "test",
				Channel: 0, Program: &p, Volume: 100, Pan: 64,
				Enabled: true, IsCoreTrack: false,
				Events: []schema.NoteEvent{
					{Type: "note", Pitch: 60, StartBeat: 0, DurationBeat: 1, Velocity: 100},
				},
			},
		},
	}

	out := filepath.Join(dir, "render.mid")
	result, err := midi.RenderMIDI(mid, out, nil)
	if err != nil {
		t.Fatal(err)
	}

	midiData, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}

	record, err := fs.SaveFile("roundtrip", midiData, result)
	if err != nil {
		t.Fatal(err)
	}

	// Load back and verify.
	loadedData, loadedRecord, err := fs.LoadFile("roundtrip")
	if err != nil {
		t.Fatal(err)
	}

	if len(loadedData) != len(midiData) {
		t.Errorf("size mismatch: %d vs %d", len(loadedData), len(midiData))
	}
	if loadedRecord.FileSize != record.FileSize {
		t.Errorf("FileSize mismatch: %d vs %d", loadedRecord.FileSize, record.FileSize)
	}
	if loadedRecord.RenderMeta.TotalNoteEvents != 1 {
		t.Errorf("TotalNoteEvents = %d, want 1", loadedRecord.RenderMeta.TotalNoteEvents)
	}
}

func TestEmptyBaseDir(t *testing.T) {
	// ListFiles should handle a non-existent directory gracefully.
	dir := filepath.Join(t.TempDir(), "nonexistent")
	fs := NewFileStore(dir)

	records, err := fs.ListFiles()
	if err != nil {
		t.Fatalf("ListFiles on nonexistent dir: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 files, got %d", len(records))
	}
}

func TestSaveFile_Overwrites(t *testing.T) {
	dir := t.TempDir()
	fs := NewFileStore(dir)

	fs.SaveFile("dup", []byte("first"), nil)
	fs.SaveFile("dup", []byte("second"), nil)

	data, _, err := fs.LoadFile("dup")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "second" {
		t.Errorf("got %q, want %q", string(data), "second")
	}
}
