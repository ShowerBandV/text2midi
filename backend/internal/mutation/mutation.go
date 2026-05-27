// Package mutation provides a music-theory-driven mutation engine that applies
// random transformations to note events, ensuring each generation is unique.
// Apply() runs a chain of mutation rules with configurable probabilities.
package mutation

import (
	"math/rand"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// Rule defines a single mutation operation.
type Rule struct {
	Name    string
	Chance  float64 // probability this rule fires (0.0-1.0)
	ApplyFn func(events []schema.NoteEvent, scale []int, barCount int) []schema.NoteEvent
}

// Engine holds the mutation rule chain.
type Engine struct {
	rules []Rule
	rng   *rand.Rand
}

// NewEngine creates a mutation engine with the standard rule set.
func NewEngine(seed int64) *Engine {
	return &Engine{
		rng: rand.New(rand.NewSource(seed)),
		rules: []Rule{
			{
				Name:   "octave_shift",
				Chance: 0.15,
				ApplyFn: func(events []schema.NoteEvent, scale []int, barCount int) []schema.NoteEvent {
					// Randomly shift some notes up/down an octave.
					for i := range events {
						if rand.Float64() < 0.25 {
							if rand.Float64() < 0.5 {
								events[i].Pitch = clamp(events[i].Pitch+12, 21, 108)
							} else {
								events[i].Pitch = clamp(events[i].Pitch-12, 21, 108)
							}
						}
					}
					return events
				},
			},
			{
				Name:   "rhythm_swing",
				Chance: 0.25,
				ApplyFn: func(events []schema.NoteEvent, scale []int, barCount int) []schema.NoteEvent {
					// Slightly offset even 16th-notes for swing feel.
					for i := range events {
						step := int(events[i].StartBeat*4 + 0.5) % 2
						if step == 0 && rand.Float64() < 0.4 {
							events[i].StartBeat += 0.06 // delay by ~1/64 note
						}
					}
					return events
				},
			},
			{
				Name:   "rest_insertion",
				Chance: 0.20,
				ApplyFn: func(events []schema.NoteEvent, scale []int, barCount int) []schema.NoteEvent {
					// Remove some notes to create rests.
					filtered := make([]schema.NoteEvent, 0, len(events))
					for _, e := range events {
						if rand.Float64() > 0.12 {
							filtered = append(filtered, e)
						}
					}
					return filtered
				},
			},
			{
				Name:   "note_reorder",
				Chance: 0.10,
				ApplyFn: func(events []schema.NoteEvent, scale []int, barCount int) []schema.NoteEvent {
					// Swap adjacent notes within the same bar.
					barSize := 4.0
					for bar := 0; bar < barCount; bar++ {
						start := float64(bar) * barSize
						end := start + barSize
						// Collect notes in this bar.
						var indices []int
						for i := range events {
							if events[i].StartBeat >= start && events[i].StartBeat < end {
								indices = append(indices, i)
							}
						}
						// Swap adjacent pairs.
						for i := 0; i < len(indices)-1; i += 2 {
							if rand.Float64() < 0.5 {
								events[indices[i]], events[indices[i+1]] = events[indices[i+1]], events[indices[i]]
							}
						}
					}
					return events
				},
			},
			{
				Name:   "transposition",
				Chance: 0.12,
				ApplyFn: func(events []schema.NoteEvent, scale []int, barCount int) []schema.NoteEvent {
					// Transpose a random section up/down by 2-5 semitones.
					if len(events) == 0 {
						return events
					}
					startIdx := rand.Intn(len(events))
					endIdx := startIdx + rand.Intn(len(events)-startIdx)
					if endIdx > len(events) {
						endIdx = len(events)
					}
					trans := []int{-5, -4, -3, -2, 2, 3, 4, 5}[rand.Intn(8)]
					for i := startIdx; i < endIdx; i++ {
						events[i].Pitch = clamp(events[i].Pitch+trans, 21, 108)
					}
					return events
				},
			},
			{
				Name:   "velocity_rewrite",
				Chance: 0.30,
				ApplyFn: func(events []schema.NoteEvent, scale []int, barCount int) []schema.NoteEvent {
					// Assign fresh velocities with a dynamic arc across bars.
					for i := range events {
						bar := int(events[i].StartBeat) / 4
						if bar >= barCount {
							bar = barCount - 1
						}
						// Dynamic arc: louder in middle bars, softer at ends.
						arc := 0.6 + 0.8*float64(bar)/float64(barCount)
						if bar > barCount/2 {
							arc = 0.6 + 0.8*(1.0-float64(bar)/float64(barCount))
						}
						if arc < 0.3 {
							arc = 0.3
						}
						base := int(80 * arc)
						events[i].Velocity = clamp(base+rand.Intn(20)-10, 1, 127)
					}
					return events
				},
			},
		},
	}
}

// Apply runs all mutation rules on the given events map in place.
// scale provides the key's scale pitches (MIDI) for pitch-aware rules.
// fv optionally adjusts rule probabilities per style (zero-value = default).
func (e *Engine) Apply(eventsByTrack map[string][]schema.NoteEvent, scale []int, barCount int, fv schema.FeatureVector) {
	for _, rule := range e.rules {
		chance := rule.Chance

		// Adjust rule probability based on feature vector.
		switch rule.Name {
		case "rest_insertion":
			// High energy styles (rock/metal) should NOT lose notes randomly.
			chance *= (1.0 - fv.Energy*0.8)
			// Lo-fi benefits from sparser feel.
			chance *= (1.0 + fv.LoFi*0.5)
		case "note_reorder":
			// Reduce for high energy (rock needs steady rhythm).
			chance *= (1.0 - fv.Energy*0.8)
		case "rhythm_swing":
			// More swing for lo-fi, less for high-energy rock.
			chance *= (1.0 + fv.LoFi*0.8 - fv.Energy*0.5)
		case "octave_shift":
			// More shifts for dense arrangements.
			chance *= (1.0 + fv.Density*0.5)
		case "transposition":
			// Less transposition for high-tension (blues/rock --keep key stable).
			chance *= (1.0 - fv.Tension*0.6)
		case "velocity_rewrite":
			// Always useful for expression; keep high.
			if chance < 0.5 {
				chance = 0.5
			}
		}
		if chance < 0.01 {
			chance = 0.01
		}
		if chance > 0.95 {
			chance = 0.95
		}

		if e.rng.Float64() < chance {
			for trackID := range eventsByTrack {
				if trackID == "drums" || len(eventsByTrack[trackID]) == 0 {
					continue // skip drums --rhythmic mutations don't apply well
				}
				eventsByTrack[trackID] = rule.ApplyFn(eventsByTrack[trackID], scale, barCount)
			}
		}
	}
}

// BuildScaleFromKey returns MIDI pitches for a given key (e.g. "C minor").
func BuildScaleFromKey(root, mode string) []int {
	semi := map[string]int{
		"C": 0, "C#": 1, "D": 2, "D#": 3, "E": 4, "F": 5,
		"F#": 6, "G": 7, "G#": 8, "A": 9, "A#": 10, "B": 11,
	}
	var intervals []int
	switch mode {
	case "minor", "natural_minor":
		intervals = []int{0, 2, 3, 5, 7, 8, 10}
	case "major":
		intervals = []int{0, 2, 4, 5, 7, 9, 11}
	default:
		intervals = []int{0, 2, 3, 5, 7, 8, 10}
	}
	r, ok := semi[root]
	if !ok {
		r = 0
	}
	// Generate pitches across 3 octaves (MIDI 48-96).
	var pitches []int
	for oct := 3; oct <= 5; oct++ {
		base := (oct + 1) * 12
		for _, iv := range intervals {
			p := base + r + iv
			if p >= 21 && p <= 108 {
				pitches = append(pitches, p)
			}
		}
	}
	return pitches
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
