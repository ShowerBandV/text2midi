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

	hook := buildHook(scale, rng)           // 4-note motif
	answer := buildAnswer(hook, scale, rng) // related but different

	for bar := 0; bar < totalBars; bar++ {
		base := float64(bar) * 4.0
		cycle := bar % 16 // 16-bar phrase cycle

		var phrase []int
		var spacing float64

		switch {
		case cycle < 4:
			// Statement: play the hook as-is, with breath between phrases.
			phrase = hook
			spacing = 0.5
			if bar%2 == 1 {
				continue // rest every other bar for breathing
			}
		case cycle < 8:
			// Transposition: hook up a 4th.
			phrase = transposeMotif(hook, 5)
			spacing = 0.5
			if bar%2 == 1 {
				continue
			}
		case cycle < 12:
			// Rhythmic variation: same notes, double-time feel.
			phrase = hook
			spacing = 0.25
		case cycle < 16:
			// Answer: new phrase that resolves.
			phrase = answer
			spacing = 0.5
			if bar%2 == 1 {
				continue
			}
		}

		for i, p := range phrase {
			oct := 4
			if bar >= 16 {
				oct = 5 // second half of song: octave up
			}
			pitch := p + 12*oct
			if pitch < 48 {
				pitch += 12
			}
			if pitch > 96 {
				pitch -= 12
			}
			vel := 100
			dur := spacing * 0.8
			if dur < 0.08 {
				dur = 0.08
			}
			events = append(events, schema.NoteEvent{
				Type: "note", Pitch: pitch,
				StartBeat: base + float64(i)*spacing, DurationBeat: dur,
				Velocity: vel,
			})
		}
	}

	return events
}

// buildHook creates a 4-note motif from the scale.
func buildHook(scale []int, rng *rand.Rand) []int {
	if len(scale) < 4 {
		return []int{0, 2, 4, 2}
	}
	// Pick notes that form a recognizable shape: up-down or down-up.
	idx := rng.Intn(len(scale) / 2)
	hook := []int{
		scale[idx],
		scale[(idx+2)%len(scale)],
		scale[(idx+4)%len(scale)],
		scale[(idx+1)%len(scale)],
	}
	return hook
}

// buildAnswer creates a 4-note phrase related to the hook but resolving differently.
func buildAnswer(hook []int, scale []int, rng *rand.Rand) []int {
	// Reverse direction and end on root.
	answer := []int{hook[2], hook[1], hook[0], scale[0]}
	return answer
}

// transposeMotif shifts all notes by N semitones.
func transposeMotif(motif []int, semitones int) []int {
	result := make([]int, len(motif))
	for i, p := range motif {
		result[i] = p + semitones
	}
	return result
}
