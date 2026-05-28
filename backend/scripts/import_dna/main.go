package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ShowerBandV/text2midi/internal/midi"
	"github.com/ShowerBandV/text2midi/internal/musicdna"
	"github.com/ShowerBandV/text2midi/internal/schema"
)

func main() {
	srcDir := "../DNA/jaychou"
	if len(os.Args) > 1 {
		srcDir = os.Args[1]
	}
	targetDir := "./templates"
	if len(os.Args) > 2 {
		targetDir = os.Args[2]
	}

	templateLib := musicdna.NewTemplateDB(targetDir)

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read dir %s: %v\n", srcDir, err)
		os.Exit(1)
	}

	imported := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".mid") &&
			!strings.HasSuffix(strings.ToLower(entry.Name()), ".midi") {
			continue
		}

		path := filepath.Join(srcDir, entry.Name())
		fmt.Printf("\nProcessing: %s\n", entry.Name())

		parsedNotes, totalBars, _, err := midi.ReadMIDIFile(path)
		if err != nil {
			fmt.Printf("  skip: %v\n", err)
			continue
		}
		if len(parsedNotes) == 0 {
			fmt.Printf("  skip: no notes\n")
			continue
		}

		ticksPerBeat := 480
		beatNotes := midi.ConvertTicksToBeats(parsedNotes, ticksPerBeat)

		// Group notes by track index for extractor.
		type trackInfo struct {
			notes    []schema.NoteEvent
			avgPitch float64
		}
		rawTracks := make(map[int]*trackInfo)
		for _, bn := range beatNotes {
			if rawTracks[bn.TrackIndex] == nil {
				rawTracks[bn.TrackIndex] = &trackInfo{}
			}
			ti := rawTracks[bn.TrackIndex]
			ti.notes = append(ti.notes, schema.NoteEvent{
				Type: "note", Pitch: bn.Pitch,
				StartBeat: bn.StartBeat, DurationBeat: bn.Duration, Velocity: bn.Velocity,
			})
			ti.avgPitch += float64(bn.Pitch)
		}

		// Assign lead track: highest average pitch with sufficient notes.
		eventsByTrack := make(map[string][]schema.NoteEvent)
		bestLeadScore := 0.0
		bestLeadTrack := -1
		for ti, info := range rawTracks {
			if len(info.notes) > 0 {
				info.avgPitch /= float64(len(info.notes))
			}
			// Melodic tracks: avg pitch 60-84, at least 10 notes.
			if info.avgPitch > 60 && info.avgPitch < 85 && len(info.notes) >= 10 {
				score := float64(len(info.notes)) * (info.avgPitch - 55)
				if score > bestLeadScore {
					bestLeadScore = score
					bestLeadTrack = ti
				}
			}
		}

		for ti, info := range rawTracks {
			trackID := fmt.Sprintf("track_%d", ti)
			if ti == bestLeadTrack {
				trackID = "lead"
			}
			eventsByTrack[trackID] = info.notes
		}

		// Also create a "chords" track by detecting chord tones.
		// Chord detection: group simultaneous notes within 0.1 beat.
		type chordGroup struct {
			beat   float64
			pitches []int
		}
		var chords []chordGroup
		for _, bn := range beatNotes {
			found := false
			for i, cg := range chords {
				if abs(cg.beat-bn.StartBeat) < 0.1 {
					chords[i].pitches = append(chords[i].pitches, bn.Pitch%12)
					found = true
					break
				}
			}
			if !found {
				chords = append(chords, chordGroup{beat: bn.StartBeat, pitches: []int{bn.Pitch % 12}})
			}
		}

		// Filter to groups with 3+ different pitch classes (likely chords).
		for _, cg := range chords {
			unique := make(map[int]bool)
			for _, p := range cg.pitches {
				unique[p] = true
			}
			if len(unique) >= 3 {
				for p := range unique {
					eventsByTrack["chords"] = append(eventsByTrack["chords"], schema.NoteEvent{
						Type: "note", Pitch: 60 + p,
						StartBeat: cg.beat, DurationBeat: 0.5, Velocity: 70,
					})
				}
			}
		}

		name := strings.TrimSuffix(entry.Name(), ".mid")
		name = strings.TrimSuffix(name, ".midi")

		_, err = templateLib.FromMIDI(eventsByTrack, totalBars, "C major", "jaychou", name, path)
		if err != nil {
			fmt.Printf("  save error: %v\n", err)
			continue
		}
		imported++
	}

	fmt.Printf("\nDone: %d/%d files imported to %s\n", imported, len(entries), targetDir)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
