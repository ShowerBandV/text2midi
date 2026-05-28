// Package generator — FX/special effects track generation.
package generator

import (
	"github.com/ShowerBandV/text2midi/internal/schema"
)

// GenerateFX creates special effects events (risers, sweeps, impacts).
// These add production value and transition cues to the arrangement.
func GenerateFX(bars int, bpm int, energy float64, tension float64) []schema.NoteEvent {
	var events []schema.NoteEvent

	if bars < 2 || energy < 0.2 {
		return events
	}

	bpb := 4.0
	totalBeats := float64(bars) * bpb
	rng := newGlobalRand()

	// 1. White noise sweep on the last beat of each section (if energy > 0.4).
	if energy > 0.4 {
		sweepStart := totalBeats - 1.0
		for i := 0; i < 4; i++ {
			pitch := 30 + rng.Intn(20)
			vel := 40 + rng.Intn(30)
			events = append(events, schema.NoteEvent{
				Pitch: pitch, StartBeat: sweepStart + float64(i)*0.25,
				DurationBeat: 0.25, Velocity: vel,
			})
		}
	}

	// 2. Sub-bass hit on bar 0 (impact).
	events = append(events, schema.NoteEvent{
		Pitch: 28, StartBeat: 0, DurationBeat: 0.5, Velocity: 90,
	})

	// 3. Midpoint riser (increasing pitch + velocity).
	if bars >= 4 && energy > 0.3 {
		midPoint := totalBeats / 2.0
		for i := 0; i < 8; i++ {
			progress := float64(i) / 8.0
			pitch := 40 + int(progress*30)
			vel := 50 + int(progress*50)
			events = append(events, schema.NoteEvent{
				Pitch: pitch, StartBeat: midPoint + progress*2.0,
				DurationBeat: 0.125 + progress*0.125, Velocity: vel,
			})
		}
	}

	// 4. Tension cluster (dissonant cluster near climax).
	if tension > 0.6 {
		for _, p := range []int{30, 33, 36, 39} {
			events = append(events, schema.NoteEvent{
				Pitch: p, StartBeat: totalBeats - 1.5,
				DurationBeat: 1.5, Velocity: int(40 + tension*30),
			})
		}
	}

	// 5. Reverse cymbal swell before bar 0 (if enough energy).
	if energy > 0.5 {
		for i := 0; i < 6; i++ {
			progress := float64(i) / 6.0
			vel := int(progress * 80)
			events = append(events, schema.NoteEvent{
				Pitch: 42 + int(progress*8),
				StartBeat: -1.0 + progress*1.0,
				DurationBeat: 0.5 + progress*0.5,
				Velocity: vel,
			})
		}
	}

	return events
}

// GenerateImpact creates a one-shot impact/stinger for transitions.
func GenerateImpact(bar int, intensity float64) []schema.NoteEvent {
	beat := float64(bar) * 4.0
	vel := int(60 + intensity*50)
	return []schema.NoteEvent{
		{Pitch: 25, StartBeat: beat, DurationBeat: 0.5, Velocity: vel},
		{Pitch: 32, StartBeat: beat, DurationBeat: 0.5, Velocity: vel - 10},
		{Pitch: 37, StartBeat: beat, DurationBeat: 0.3, Velocity: vel - 20},
		{Pitch: 44, StartBeat: beat + 0.25, DurationBeat: 0.25, Velocity: vel - 15},
	}
}

// newGlobalRand returns a deterministic RNG for FX generation.
func newGlobalRand() *globalRand {
	return &globalRand{seed: 42}
}

type globalRand struct {
	seed int
}

func (g *globalRand) Intn(n int) int {
	g.seed = (g.seed*1103515245 + 12345) & 0x7fffffff
	return g.seed % n
}
