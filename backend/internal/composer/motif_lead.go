package composer

import (
	"math/rand"
	"strings"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// GenerateLeadMotif creates chord-aware melody with question-answer phrasing.
func GenerateLeadMotif(scale []int, totalBars int, _ float64) []schema.NoteEvent {
	return GenerateLeadMotifWithChords(scale, nil, totalBars)
}

// GenerateLeadMotifWithChords creates chord-aware melody.
// Strong beats land on chord tones (root/3rd/5th); weak beats use scale passing tones.
func GenerateLeadMotifWithChords(scale []int, chords []string, totalBars int) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(globalSeed))
	var events []schema.NoteEvent

	slow := []float64{0.5, 0.5, 0.5, 0.5}
	fast := []float64{0.25, 0.25, 0.25, 0.5, 0.25, 0.25, 0.5}

	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0
		cycle := bar % 16

		chordRoot := 0
		if chords != nil && bar < len(chords) {
			chordRoot = chordRootMIDIPiano(chords[bar], 0)
		}

		var noteCount int
		var rhythm []float64
		octave := 4

		switch {
		case cycle < 4:
			noteCount = 4
			rhythm = slow
			if bar%2 == 1 { continue }
		case cycle < 8:
			noteCount = 7
			rhythm = fast
			octave = 5
		case cycle < 12:
			noteCount = 8
			rhythm = fast
			octave = 5
		default:
			noteCount = 3
			rhythm = slow
			octave = 4
			if bar%2 == 1 { continue }
		}

		var t float64
		for i := 0; i < noteCount; i++ {
			var pitch int
			if i%2 == 0 && chordRoot > 0 && chords != nil {
				// Strong beat: chord tone.
				offsets := []int{0, 4, 7} // root, maj3, 5th
				if strings.Contains(chords[bar%len(chords)], "m") {
					offsets[1] = 3 // min3
				}
				pitch = chordRoot + offsets[rng.Intn(3)]
			} else {
				// Weak beat: relative interval from chord root.
				intervals := []int{0, 2, 4, 5, 7, 9, 11}
				if chordRoot > 0 {
					pitch = chordRoot + intervals[rng.Intn(7)]
				} else {
					pitch = intervals[rng.Intn(7)] // no chord: just use interval
				}
			}

			pitch += 12 * octave
			if pitch < 48 { pitch += 12 }
			if pitch > 96 { pitch -= 12 }

			dur := rhythm[i%len(rhythm)] * 0.8
			if dur < 0.06 { dur = 0.06 }

			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: pitch,
				StartBeat: base + t, DurationBeat: dur,
				Velocity: 100,
			})
			t += rhythm[i%len(rhythm)]
		}
	}

	return events
}
