// Package composer — Groove quantization engine.
// Applies swing/shuffle feel to straight MIDI patterns.
package composer

import (
	"fmt"
	"math/rand"

	"github.com/yourname/text2midi/internal/schema"
)

// GrooveSettings controls the quantization feel.
// SwingType defines the swing grid.
type SwingType int

const (
	SwingStraight     SwingType = iota // 0: no swing
	SwingTriplet                       // 1: true triplet swing (66.6%)
	SwingMPC                           // 2: MPC swing (54-64%, user-selectable)
	SwingShuffle                       // 3: shuffle feel (50-58%)
)

// GrooveSettings controls the quantization feel.
type GrooveSettings struct {
	SwingType  SwingType // which swing grid
	SwingAmount float64  // 0.0-1.0: how much swing within the grid
	// Named presets (set SwingType and SwingAmount automatically):
	PresetName string   // "mpc62", "triplet", "shuffle55", "straight"
	Humanize   float64  // 0.0=robotic, 1.0=loose
	AccentBeat float64  // 0.0-1.0 how much to accent beat 1
}

// ParsePreset sets SwingType and SwingAmount from a preset name string.
// Examples: "mpc62" = MPC swing at 62%, "triplet" = true triplet, "shuffle55" = shuffle at 55%
func (gs *GrooveSettings) ParsePreset(preset string) {
	switch {
	case preset == "straight" || preset == "":
		gs.SwingType = SwingStraight
		gs.SwingAmount = 0
	case preset == "triplet":
		gs.SwingType = SwingTriplet
		gs.SwingAmount = 1.0
	case len(preset) >= 3 && preset[:3] == "mpc":
		gs.SwingType = SwingMPC
		amount := 0.58 // default
		if len(preset) > 3 {
			var pct float64
			if _, err := fmt.Sscanf(preset[3:], "%f", &pct); err == nil {
				amount = pct / 100.0
			}
		}
		if amount < 0.50 {
			amount = 0.50
		}
		if amount > 0.66 {
			amount = 0.66
		}
		gs.SwingAmount = amount
	case len(preset) >= 7 && preset[:7] == "shuffle":
		gs.SwingType = SwingShuffle
		amount := 0.55
		if len(preset) > 7 {
			var pct float64
			if _, err := fmt.Sscanf(preset[7:], "%f", &pct); err == nil {
				amount = pct / 100.0
			}
		}
		gs.SwingAmount = amount
	}
}

// DefaultGroove returns reasonable groove settings for a given energy level.
func DefaultGroove(energy float64, preset string) GrooveSettings {
	gs := GrooveSettings{
		Humanize:   0.05 + energy*0.1,
		AccentBeat: 0.6 + energy*0.3,
	}
	gs.ParsePreset(preset)
	return gs
}

// ApplyGroove modifies note timing to add swing/shuffle feel.
// For 8th-note pairs: delays the offbeat 8th by a swing ratio.
func ApplyGroove(events []schema.NoteEvent, gs GrooveSettings) []schema.NoteEvent {
	if len(events) == 0 {
		return events
	}

	for i := range events {
		e := &events[i]

		// Find the beat position within the bar (0-3.999).
		barStart := float64(int(e.StartBeat/4)) * 4
		beatPos := e.StartBeat - barStart
		beatFraction := beatPos - float64(int(beatPos))

		// Apply swing based on swing type.
		var swingOffset float64
		switch gs.SwingType {
		case SwingTriplet:
			// True triplet: delay offbeat 8ths by 1/6 of a beat.
			if beatFraction > 0.4 && beatFraction < 0.6 {
				swingOffset = 0.167 * gs.SwingAmount // 1/6 beat
			}
		case SwingMPC:
			// MPC swing: delay offbeat 16th notes by percentage.
			// At 62%, offbeat 16th is 62% of the way to triplet position.
			if beatFraction > 0.45 && beatFraction < 0.55 {
				swingOffset = 0.083 * gs.SwingAmount // 50-66% of 1/12
			}
			if beatFraction > 0.2 && beatFraction < 0.3 {
				swingOffset = 0.083 * gs.SwingAmount * 0.5
			}
			if beatFraction > 0.7 && beatFraction < 0.8 {
				swingOffset = 0.083 * gs.SwingAmount * 0.5
			}
		case SwingShuffle:
			// Shuffle: delay offbeat 8ths like triplet but less extreme.
			if beatFraction > 0.4 && beatFraction < 0.6 {
				swingOffset = 0.1 * gs.SwingAmount // ~50% triplet
			}
		}
		e.StartBeat += swingOffset

		// Humanize: random micro-timing variation (not applied to all notes).
		if rand.Float64() < gs.Humanize*2 {
			jitter := (rand.Float64() - 0.5) * gs.Humanize * 0.06
			e.StartBeat += jitter
		}

		// Accent beat 1.
		if beatPos < 0.25 {
			e.Velocity = int(float64(e.Velocity) * (1.0 + gs.AccentBeat*0.15))
			if e.Velocity > 127 {
				e.Velocity = 127
			}
		}

		// Prevent negative start beat.
		if e.StartBeat < 0 {
			e.StartBeat = 0
		}
	}

	// Apply phrase-end rubato: slightly delay and stretch notes at end of 4-bar phrases.
	ApplyRubato(events)
	return events
}

// ApplyRubato applies subtle tempo variations at phrase boundaries.
// Last beat of each 4-bar phrase gets a slight delay (breathe feel).
func ApplyRubato(events []schema.NoteEvent) {
	if len(events) == 0 {
		return
	}

	phraseLen := 4 // bars
	processed := make(map[int]bool) // track which notes we've stretched

	for i := range events {
		bar := int(events[i].StartBeat) / 4
		// Check if this note is in the last beat of a phrase-ending bar.
		isPhraseEnd := (bar+1)%phraseLen == 0 && bar > 0
		if !isPhraseEnd {
			continue
		}

		beatInBar := events[i].StartBeat - float64(bar)*4
		if beatInBar < 3.0 || processed[i] {
			continue
		}

		// Apply rubato: slight delay (~15ms) and duration stretch.
		events[i].StartBeat += 0.03
		events[i].DurationBeat *= 1.08
		processed[i] = true
	}
}
