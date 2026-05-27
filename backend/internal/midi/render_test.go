package midi

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// testMidiIR builds a minimal MidiIR for testing.
func testMidiIR() schema.MidiIR {
	p := 0
	return schema.MidiIR{
		Meta: schema.Meta{
			Title: "Test", BPM: 120, TicksPerBeat: 480,
			TimeSignature: schema.TimeSignature{Numerator: 4, Denominator: 4},
			KeySignature:  "C minor", TotalBars: 4,
			BeatsPerBar: 4, TotalBeats: 16, Loopable: true,
		},
		Tracks: []schema.TrackIR{
			{
				ID: "test", Name: "TestTrack", Role: "test",
				Channel: 0, Program: &p, Volume: 100, Pan: 64,
				Enabled: true, IsCoreTrack: false,
				Events: []schema.NoteEvent{
					{Type: "note", Pitch: 60, StartBeat: 0, DurationBeat: 1, Velocity: 100},
					{Type: "note", Pitch: 64, StartBeat: 1, DurationBeat: 0.5, Velocity: 90},
					{Type: "note", Pitch: 67, StartBeat: 2, DurationBeat: 2, Velocity: 80},
				},
			},
		},
	}
}

func TestRenderMIDI_CreatesFile(t *testing.T) {
	mid := testMidiIR()
	out := filepath.Join(t.TempDir(), "test.mid")

	result, err := RenderMIDI(mid, out, nil)
	if err != nil {
		t.Fatalf("RenderMIDI failed: %v", err)
	}

	if result.OutputPath != out {
		t.Errorf("OutputPath = %q, want %q", result.OutputPath, out)
	}
	if result.TotalTracks != 1 {
		t.Errorf("TotalTracks = %d, want 1", result.TotalTracks)
	}
	if result.TotalNoteEvents != 3 {
		t.Errorf("TotalNoteEvents = %d, want 3", result.TotalNoteEvents)
	}
	if result.DurationSeconds <= 0 {
		t.Errorf("DurationSeconds = %f, want > 0", result.DurationSeconds)
	}

	// Verify file exists and has content.
	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat output: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("output file is empty")
	}
}

func TestRenderMIDI_HeaderStructure(t *testing.T) {
	mid := testMidiIR()
	out := filepath.Join(t.TempDir(), "header_test.mid")
	_, err := RenderMIDI(mid, out, nil)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}

	// Check "MThd" magic
	if string(data[0:4]) != "MThd" {
		t.Fatalf("header magic = %q, want \"MThd\"", data[0:4])
	}

	// Header length = 6
	headerLen := binary.BigEndian.Uint32(data[4:8])
	if headerLen != 6 {
		t.Errorf("header length = %d, want 6", headerLen)
	}

	// Format = 1 (multi-track)
	format := binary.BigEndian.Uint16(data[8:10])
	if format != 1 {
		t.Errorf("format = %d, want 1", format)
	}

	// nTracks: 1 meta + 1 instrument = 2
	nTracks := binary.BigEndian.Uint16(data[10:12])
	if nTracks != 2 {
		t.Errorf("nTracks = %d, want 2", nTracks)
	}

	// Ticks per beat
	tpb := binary.BigEndian.Uint16(data[12:14])
	if tpb != 480 {
		t.Errorf("ticks_per_beat = %d, want 480", tpb)
	}
}

func TestRenderMIDI_TrackChunks(t *testing.T) {
	mid := testMidiIR()
	out := filepath.Join(t.TempDir(), "tracks_test.mid")
	_, err := RenderMIDI(mid, out, nil)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}

	// After 14-byte header, there should be "MTrk" markers.
	mtrkCount := 0
	for i := 14; i < len(data)-4; i++ {
		if string(data[i:i+4]) == "MTrk" {
			mtrkCount++
		}
	}
	if mtrkCount != 2 {
		t.Errorf("found %d MTrk chunks, want 2 (meta + 1 instrument)", mtrkCount)
	}
}

func TestRenderMIDI_MetaTrackContent(t *testing.T) {
	mid := testMidiIR()
	out := filepath.Join(t.TempDir(), "meta_test.mid")
	_, err := RenderMIDI(mid, out, nil)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}

	// Meta track starts at offset 14, after 14-byte header.
	// Check it starts with "MTrk".
	if string(data[14:18]) != "MTrk" {
		t.Fatalf("expected MTrk at offset 14")
	}

	// After MTrk + 4-byte length, the first events should be:
	// delta=0 FF 51 03 (set tempo)
	metaStart := 22 // 14 header + 8 MTrk chunk header
	if data[metaStart] != 0x00 {
		t.Errorf("meta track first delta = %02x, want 00", data[metaStart])
	}
	if data[metaStart+1] != 0xFF || data[metaStart+2] != 0x51 || data[metaStart+3] != 0x03 {
		t.Errorf("expected set_tempo meta event at offset %d, got %02x %02x %02x %02x",
			metaStart, data[metaStart+1], data[metaStart+2], data[metaStart+3], data[metaStart+4])
	}
}

func TestRenderMIDI_SelectedTracks(t *testing.T) {
	mid := testMidiIR()
	// Add a second track.
	p := 0
	mid.Tracks = append(mid.Tracks, schema.TrackIR{
		ID: "extra", Name: "Extra", Role: "extra",
		Channel: 1, Program: &p, Volume: 100, Pan: 64,
		Enabled: true, IsCoreTrack: false,
		Events: []schema.NoteEvent{
			{Type: "note", Pitch: 72, StartBeat: 0, DurationBeat: 1, Velocity: 100},
		},
	})

	out := filepath.Join(t.TempDir(), "select_test.mid")
	result, err := RenderMIDI(mid, out, []string{"test"})
	if err != nil {
		t.Fatalf("RenderMIDI with selection failed: %v", err)
	}
	if result.TotalTracks != 1 {
		t.Errorf("with selection: TotalTracks = %d, want 1", result.TotalTracks)
	}
	if result.TotalNoteEvents != 3 {
		t.Errorf("with selection: TotalNoteEvents = %d, want 3", result.TotalNoteEvents)
	}
}

func TestRenderMIDI_InvalidTrack(t *testing.T) {
	mid := testMidiIR()
	out := filepath.Join(t.TempDir(), "invalid_test.mid")
	_, err := RenderMIDI(mid, out, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent selected track")
	}
}

func TestRenderMIDI_EmptySelection(t *testing.T) {
	mid := testMidiIR()
	out := filepath.Join(t.TempDir(), "empty_test.mid")
	_, err := RenderMIDI(mid, out, []string{})
	if err == nil {
		t.Fatal("expected error for empty selection")
	}
}

func TestBeatToTick(t *testing.T) {
	tests := []struct {
		beat float64
		tpb  int
		want int
	}{
		{0, 480, 0},
		{1, 480, 480},
		{4, 480, 1920},
		{0.5, 480, 240},
		{0.25, 480, 120},
	}
	for _, tt := range tests {
		got := beatToTick(tt.beat, tt.tpb)
		if got != tt.want {
			t.Errorf("beatToTick(%f, %d) = %d, want %d", tt.beat, tt.tpb, got, tt.want)
		}
	}
}

func TestBPMToTempo(t *testing.T) {
	tests := []struct {
		bpm  int
		want int
	}{
		{120, 500000},
		{140, 428571},
		{60, 1000000},
	}
	for _, tt := range tests {
		got := bpmToTempo(tt.bpm)
		if got != tt.want {
			t.Errorf("bpmToTempo(%d) = %d, want %d", tt.bpm, got, tt.want)
		}
	}
}

func TestVarLen(t *testing.T) {
	tests := []struct {
		value int
		want  []byte
	}{
		{0, []byte{0x00}},
		{127, []byte{0x7F}},
		{128, []byte{0x81, 0x00}},
		{16383, []byte{0xFF, 0x7F}},
		{16384, []byte{0x81, 0x80, 0x00}},
	}
	for _, tt := range tests {
		got := encodeVarLen(tt.value)
		if len(got) != len(tt.want) {
			t.Errorf("encodeVarLen(%d) = %v (len=%d), want %v", tt.value, got, len(got), tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("encodeVarLen(%d)[%d] = %02x, want %02x", tt.value, i, got[i], tt.want[i])
			}
		}
	}
}
