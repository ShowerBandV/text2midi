// Package composer — Stem Exporter.
// Splits a multi-track MIDI into separate .mid files per instrument group.
// Game developers use stems for Wwise/FMOD integration.
package composer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ShowerBandV/text2midi/internal/midi"
	"github.com/ShowerBandV/text2midi/internal/schema"
)

// StemGroup defines which tracks go into a stem.
type StemGroup struct {
	Name   string   // filename stem (e.g. "drums", "bass")
	Tracks []string // track IDs to include
}

// DefaultStemGroups is the standard stem split.
var DefaultStemGroups = []StemGroup{
	{Name: "drums", Tracks: []string{"drums", "hihat", "percussion", "timpani", "taiko", "crash_cymbal"}},
	{Name: "bass", Tracks: []string{"bass"}},
	{Name: "lead", Tracks: []string{"lead", "lead_guitar", "lead_vocal", "dizi"}},
	{Name: "pads", Tracks: []string{"chords", "pad", "piano", "strings", "synth_pad", "atmosphere"}},
	{Name: "guitar", Tracks: []string{"rhythm_guitar", "rhythm_guitar", "guitar", "distorted_guitar"}},
	{Name: "fx", Tracks: []string{"counter_melody", "brass", "choir", "harp", "orchestral_hits", "guzheng", "pipa"}},
}

// ExportStems generates separate .mid files for each instrument group.
func ExportStems(midiIR schema.MidiIR, outputDir, baseName string, groups []StemGroup) error {
	if groups == nil {
		groups = DefaultStemGroups
	}

	// Build a lookup track ID → TrackIR.
	trackMap := make(map[string]schema.TrackIR)
	for _, t := range midiIR.Tracks {
		trackMap[t.Name] = t
	}

	for _, group := range groups {
		// Collect tracks for this stem group.
		var stemTracks []schema.TrackIR
		for _, trackID := range group.Tracks {
			if t, ok := trackMap[trackID]; ok {
				stemTracks = append(stemTracks, t)
			}
		}
		if len(stemTracks) == 0 {
			continue
		}

		// Build MidiIR for this stem.
		stemIR := schema.MidiIR{
			Meta:   midiIR.Meta,
			Tracks: stemTracks,
		}

		// Render to file.
		filename := fmt.Sprintf("%s_%s.mid", baseName, group.Name)
		outputPath := filepath.Join(outputDir, filename)
		result, err := midi.RenderMIDI(stemIR, outputPath, nil)
		if err != nil {
			fmt.Printf("[Stem] %s: render error: %v\n", group.Name, err)
			continue
		}
		fmt.Printf("[Stem] %s → %s (%d tracks, %d notes)\n",
			group.Name, result.OutputPath, len(stemTracks), result.TotalNoteEvents)
	}

	// Also write full mix.
	mixPath := filepath.Join(outputDir, baseName+"_full.mid")
	result, err := midi.RenderMIDI(midiIR, mixPath, nil)
	if err != nil {
		return err
	}
	fmt.Printf("[Stem] full mix → %s\n", result.OutputPath)

	return nil
}

// Exists checks if a stem export directory is ready.
func EnsureStemDir(outputPath string) error {
	stemDir := filepath.Join(filepath.Dir(outputPath), "stems")
	return os.MkdirAll(stemDir, 0755)
}
