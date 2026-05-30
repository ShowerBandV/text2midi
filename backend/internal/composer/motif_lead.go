package composer

import (
	"math/rand"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// GenerateLeadMotif creates a melody with motif development:
// statement (bars 0-3) → transposition (bars 4-7) → rhythmic variation (bars 8-11) → answer (bars 12-15).
// This gives the melody a "story" — something stated, developed, and resolved.
func GenerateLeadMotif(scale []int, totalBars int, _ float64) []schema.NoteEvent {
	rng := rand.New(rand.NewSource(globalSeed))
	var events []schema.NoteEvent

	// Build a MUSICAL phrase, not a mathematical pattern.
	// Each phrase has: question (rise) → answer (fall) → rest (breath).
	question := buildQuestion(scale, rng) // 4 notes rising
	answer := buildAnswer(scale, rng)      // 3 notes falling to resolution
	rhythmSlow := []float64{0.5, 0.5, 0.5, 0.5} // quarter-note feel
	rhythmFast := []float64{0.25, 0.25, 0.25, 0.5, 0.25, 0.25, 0.5} // eighth-note feel

	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0
		cycle := bar % 16

		var phrase []int
		var rhythm []float64
		octave := 4

		switch {
		case cycle < 4:
			// Statement: question alone, with breath.
			phrase = question
			rhythm = rhythmSlow
			if bar%2 == 1 { continue }
		case cycle < 8:
			// Development: question + answer, higher octave.
			phrase = append(append([]int{}, question...), answer...)
			rhythm = rhythmFast
			octave = 5
		case cycle < 12:
			// Climax: question repeated twice, fast, highest octave.
			phrase = append(append([]int{}, question...), question...)
			rhythm = rhythmFast
			octave = 5
		case cycle < 16:
			// Resolution: answer alone, low octave, half-time.
			phrase = answer
			rhythm = rhythmSlow
			octave = 4
			if bar%2 == 1 { continue }
		}

		var t float64
		for i, p := range phrase {
			pitch := p + 12*octave
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

// buildQuestion creates a 4-note rising phrase that ends on an "open" note (not the root).
func buildQuestion(scale []int, rng *rand.Rand) []int {
	// Walk up the scale: 1→2→3→5 (open, wants resolution)
	root := scale[0]
	third := scale[2]
	fifth := scale[4]
	if len(scale) < 5 {
		return []int{root, root + 2, root + 4, root + 7}
	}
	// Choose between two question shapes.
	if rng.Float64() < 0.5 {
		return []int{root, scale[1], third, fifth} // 1-2-3-5 (classic rising)
	}
	return []int{root, third, fifth, scale[6%len(scale)]} // 1-3-5-7 (arpeggio)
}

// buildAnswer creates a falling phrase that resolves to the root.
func buildAnswer(scale []int, rng *rand.Rand) []int {
	fifth := scale[4]
	third := scale[2]
	root := scale[0]
	if rng.Float64() < 0.5 {
		return []int{fifth, third, root} // 5-3-1 (perfect cadence)
	}
	return []int{fifth, scale[4], third, root} // 5-4-3-1 (stepwise resolution)
}
