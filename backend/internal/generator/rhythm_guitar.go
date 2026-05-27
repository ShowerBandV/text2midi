package generator

import (
	"math/rand"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// GenerateRhythmGuitar generates tight palm-muted chugging power chords.
// Metal rhythm guitar = fast 8th/16th note chugs, very short duration, heavy palm mute.
func GenerateRhythmGuitar(plan schema.SongPlan, track schema.ArrangementTrack) []schema.NoteEvent {
	totalBars := plan.TotalBars
	prog := plan.ChordProgression
	fv := plan.FeatureVector

	baseOct := 2 + int(fv.Darkness*1)
	if baseOct < 2 {
		baseOct = 2
	}
	if baseOct > 3 {
		baseOct = 3
	}

	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		chord := prog[bar%len(prog)].Chord
		rootName := chord
		if len(rootName) > 0 && rootName[len(rootName)-1] == 'm' {
			rootName = rootName[:len(rootName)-1]
		}
		rootSemi := 0
		if r, ok := map[string]int{"C": 0, "C#": 1, "D": 2, "D#": 3, "E": 4, "F": 5, "F#": 6, "G": 7, "G#": 8, "A": 9, "A#": 10, "B": 11}[rootName]; ok {
			rootSemi = r
		}

		baseMIDI := (baseOct + 1) * 12
		r := baseMIDI + rootSemi
		p5 := r + 7
		base := float64(bar) * 4.0

		// Palm-muted 8th note chugs.
		// Metal: every 8th note = one chug (root+fifth).
		// Beat 1 & 3: accent (harder pick attack).
		// Beats 2 & 4 + offbeats: lighter palm mute.
		for step := 0; step < 8; step++ {
			beat := float64(step) * 0.5
			vel := 85 + int(fv.Energy*30)
			if step%2 == 0 {
				vel += 10 // downbeat accent
			} else {
				vel -= 15 // offbeat lighter
			}
			if vel > 127 {
				vel = 127
			}
			if vel < 30 {
				vel = 30
			}

			dur := 0.05 + float64(rand.Intn(3))*0.02 // very short palm mute

			// Root + fifth chug.
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: r,
				StartBeat:    base + beat,
				DurationBeat: dur,
				Velocity:     vel,
			})
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: p5,
				StartBeat:    base + beat,
				DurationBeat: dur,
				Velocity:     vel - 3,
			})
		}
	}
	return events
}
