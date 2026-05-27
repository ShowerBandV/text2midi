package generator

import (
	"math/rand"

	"github.com/ShowerBandV/text2midi/internal/music"
	"github.com/ShowerBandV/text2midi/internal/schema"
)

// GenerateBass generates bassline events following the chord progression.
// Feature vector influences: Energy ->walking vs sustained, Darkness ->octave, Tension ->chromatic approach.
// High energy: walking 8th-note patterns (root→fifth→passing→next root).
// Low energy: long sustained roots.
func GenerateBass(plan schema.SongPlan, track schema.ArrangementTrack) []schema.NoteEvent {
	totalBars := plan.TotalBars
	prog := plan.ChordProgression
	fv := plan.FeatureVector

	// Darkness ->base octave offset.
	darkOffset := int(fv.Darkness * 6) // 0 to -6 semitones (subtler than before)

	// Tension ->chromatic approach probability.
	tensionProb := fv.Tension * 0.35

	// Decide mode: walking (high energy, rock/metal) vs sustained (low energy, ambient).
	useWalking := fv.Energy > 0.5

	// Build a fifth lookup for each chord root.
	chordFifth := make(map[int]int) // root ->fifth
	for _, cp := range prog {
		root, err := music.RootPitch(cp.Chord)
		if err != nil {
			continue
		}
		if _, ok := chordFifth[root]; !ok {
			chordFifth[root] = root + 7 // perfect fifth above
		}
	}

	var events []schema.NoteEvent
	for bar := 0; bar < totalBars; bar++ {
		chord := prog[bar%len(prog)].Chord
		root, err := music.RootPitch(chord)
		if err != nil {
			root = 36
		}
		root -= darkOffset
		if root < 24 {
			root = 24
		}
		fifth := root + 7
		if f, ok := chordFifth[root]; ok {
			fifth = f
		}

		base := float64(bar) * 4.0

		if useWalking {
			// Walking bass: 8th note feel, root→fifth patterns with chromatic approaches.
			// Pattern: root(beat 1)→fifth(beat 2)→approach(beat 3)→nextRoot(beat 4)
			// or variations depending on the beat.

			// Determine next chord's root for approach note.
			nextBar := (bar + 1) % totalBars
			nextChord := prog[nextBar%len(prog)].Chord
			nextRoot, _ := music.RootPitch(nextChord)
			nextRoot -= darkOffset
			if nextRoot < 24 {
				nextRoot = 24
			}

			// Four beats with specific pattern.
			beatPitches := [4]int{root, fifth, 0, nextRoot}

			// Beat 3: chromatic approach to next root.
			approach := nextRoot
			if rand.Float64() < tensionProb {
				// Half-step above or below for blues feel.
				approach = nextRoot + []int{-1, 1}[rand.Intn(2)]
			} else {
				// Scale step: use fifth of current chord.
				approach = fifth
			}
			if approach < 24 {
				approach = 24
			}
			if approach > 60 {
				approach = 60
			}
			beatPitches[2] = approach

			for beat, pitch := range beatPitches {
				dur := 1.0 // quarter note
				if beat == 0 || beat == 2 {
					dur = 0.9 // slight staccato for offbeats
				}
				vel := 80 + int(fv.Energy*35) + rand.Intn(10)
				if vel > 127 {
					vel = 127
				}
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: pitch,
					StartBeat:    base + float64(beat),
					DurationBeat: dur,
					Velocity:     vel,
				})
			}
		} else {
			// Sustained mode: long roots with occasional fifths.
			beatsPerBar := 4
			if fv.Energy > 0.3 {
				beatsPerBar = 2 // half notes
			}
			step := 4.0 / float64(beatsPerBar)
			for i := 0; i < beatsPerBar; i++ {
				beat := float64(i) * step
				pitch := root
				if i%2 == 1 && fv.Density > 0.3 {
					pitch = fifth
				}
				dur := step * 0.9
				vel := 65 + int(fv.Energy*25) + rand.Intn(10)
				if vel > 127 {
					vel = 127
				}
				events = append(events, schema.NoteEvent{
					Type: "note", Pitch: pitch,
					StartBeat:    base + beat,
					DurationBeat: dur,
					Velocity:     vel,
				})
			}
		}
	}
	return events
}
