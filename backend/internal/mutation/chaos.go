package mutation

import (
	"math/rand"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// ChaosConfig controls the type and intensity of creative chaos.
// These intentionally introduce "human mistakes" to break rigid perfection.
type ChaosConfig struct {
	BlueNoteChance     float64
	SuspensionChance   float64
	AnticipationChance float64
	Seed               int64
}

// DefaultChaos returns chaos settings based on tension and energy.
func DefaultChaos(tension, energy float64, seed int64) ChaosConfig {
	base := tension * 0.25
	if energy < 0.3 {
		base *= 0.5
	}
	return ChaosConfig{
		BlueNoteChance:     base * 0.3,
		SuspensionChance:   base * 0.25,
		AnticipationChance: base * 0.25,
		Seed:               seed,
	}
}

// ApplyChaos introduces creative "mistakes" to note events.
func ApplyChaos(events []schema.NoteEvent, config ChaosConfig, trackID string) []schema.NoteEvent {
	if len(events) == 0 || trackID == "drums" {
		return events
	}

	rng := rand.New(rand.NewSource(config.Seed))

	for i := range events {
		e := &events[i]

		if rng.Float64() < config.BlueNoteChance && e.Pitch > 24 {
			e.Pitch -= 1
		}

		if rng.Float64() < config.SuspensionChance {
			e.DurationBeat += 0.25
		}

		if rng.Float64() < config.AnticipationChance && e.StartBeat > 0.1 {
			e.StartBeat -= 0.1
		}
	}

	return events
}
