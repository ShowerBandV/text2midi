// Package composer — Humanizer Engine.
// Adds subtle imperfections that make MIDI sound like human performance:
// velocity drift, timing drift, ghost notes, imperfect quantization, local swing.
package composer

import (
	"math/rand"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// HumanizeConfig controls the amount and type of humanization.
type HumanizeConfig struct {
	VelocityDrift float64 // 0-1: random velocity variation (0=perfect, 1=wild)
	TimingDrift   float64 // 0-1: random timing offset (0=grid, 1=loose)
	GhostNoteRate float64 // 0-1: probability of adding ghost notes
	SwingAmount   float64 // 0-1: localized swing feel
	AccentBeat    float64 // 0-1: how much to accent beat 1
}

// DefaultHumanize returns standard humanization for a given energy/style.
func DefaultHumanize(energy, darkness float64) HumanizeConfig {
	c := HumanizeConfig{
		VelocityDrift: 0.05 + energy*0.08,
		TimingDrift:   0.02 + energy*0.04,
		GhostNoteRate: 0.05 + energy*0.03,
		SwingAmount:   0.1 + energy*0.15,
		AccentBeat:    0.6 + energy*0.3,
	}
	// Dark = more timing drift (slower, more expressive)
	if darkness > 0.6 {
		c.TimingDrift *= 1.5
		c.VelocityDrift *= 1.3
	}
	return c
}

// HumanizeEvents applies humanization to a track's events.
func HumanizeEvents(events []schema.NoteEvent, config HumanizeConfig, trackID string, rng *rand.Rand) []schema.NoteEvent {
	if len(events) == 0 || trackID == "drums" {
		return events
	}

	for i := range events {
		// 1. Velocity drift.
		drift := (rng.Float64() - 0.5) * 2 * config.VelocityDrift * 30
		events[i].Velocity += int(drift)
		if events[i].Velocity < 20 {
			events[i].Velocity = 20
		}
		if events[i].Velocity > 127 {
			events[i].Velocity = 127
		}

		// 2. Timing drift (delay).
		drift = (rng.Float64() - 0.5) * config.TimingDrift * 0.15
		events[i].StartBeat += drift
		if events[i].StartBeat < 0 {
			events[i].StartBeat = 0
		}

		// 3. Accent beat 1.
		beatInBar := events[i].StartBeat - float64(int(events[i].StartBeat/4)*4)
		if beatInBar < 0.25 {
			events[i].Velocity += int(config.AccentBeat * 15)
			if events[i].Velocity > 127 {
				events[i].Velocity = 127
			}
		}
	}

	// 4. Ghost notes: add very quiet notes on offbeats (only for melodic tracks).
	if config.GhostNoteRate > 0 && rng.Float64() < config.GhostNoteRate && len(events) > 2 {
		// Duplicate last note quietly at a slightly offset position.
		last := events[len(events)-1]
		ghost := last
		ghost.Velocity = 15 + rng.Intn(10)
		ghost.StartBeat += 0.125 + rng.Float64()*0.125
		ghost.DurationBeat = 0.05
		events = append(events, ghost)
	}

	return events
}
