package composer

import (
	"math/rand"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// InjectChaos randomly mutates a fraction of events for "happy accidents".
// prob is 0-1: fraction of events to potentially modify.
// Modifications: pitch shift ±1-3 semitones, duration stretch/shrink, velocity jitter.
func InjectChaos(events []schema.NoteEvent, prob float64) {
	if prob <= 0 || len(events) == 0 {
		return
	}
	for i := range events {
		if rand.Float64() > prob {
			continue
		}
		// Random mutation type.
		switch rand.Intn(4) {
		case 0:
			// Pitch bend: shift ±1-3 semitones.
			shift := rand.Intn(7) - 3 // -3 to +3
			events[i].Pitch += shift
			if events[i].Pitch < 21 {
				events[i].Pitch = 21
			}
			if events[i].Pitch > 108 {
				events[i].Pitch = 108
			}
		case 1:
			// Duration anomaly: stretch or shrink.
			factor := 0.5 + rand.Float64()*1.5 // 0.5x to 2x
			events[i].DurationBeat *= factor
			if events[i].DurationBeat < 0.03 {
				events[i].DurationBeat = 0.03
			}
			if events[i].DurationBeat > 8.0 {
				events[i].DurationBeat = 8.0
			}
		case 2:
			// Timing jitter: micro-shift start beat ±0.02.
			jitter := (rand.Float64() - 0.5) * 0.04
			events[i].StartBeat += jitter
			if events[i].StartBeat < 0 {
				events[i].StartBeat = 0
			}
		case 3:
			// Velocity anomaly: quiet pop or loud accent.
			if rand.Float64() < 0.5 {
				events[i].Velocity = 120
			} else {
				events[i].Velocity = 50
			}
		}
	}
}
