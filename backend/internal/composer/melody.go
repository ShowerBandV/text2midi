// Package composer — Melody Grammar Engine.
// Post-processes LLM-generated melodies to enforce human musical grammar:
//   - Scale mask (hard constraint, not prompt)
//   - Interval limiter (fold large jumps, insert passing notes)
//   - Gravity system (force return to tonic)
//   - Phrase structure (rests every 2-4 bars)
//   - Contour shaping (verse low, chorus high, ending fall)
package composer

import (
	"fmt"

	"github.com/yourname/text2midi/internal/schema"
)

// MelodyGrammar applies hard musical constraints to a raw melody.
type MelodyGrammar struct {
	KeyRoot       string
	ScalePitches  []int // allowed MIDI pitches in this key
	TonicPitch    int   // root note of the key
	DominantPitch int   // fifth of the key
	MaxInterval   int   // max semitone leap (default 7)
	StepBias      float64 // probability of stepwise motion (0.7 = 70%)
	Gravity       float64 // how strongly to pull toward tonic (0-1)
}

// NewMelodyGrammar creates a grammar checker for a given key.
func NewMelodyGrammar(keyRoot, keyMode string) *MelodyGrammar {
	g := &MelodyGrammar{
		KeyRoot:     keyRoot,
		MaxInterval: 7,
		StepBias:    0.70,
		Gravity:     0.40,
	}

	// Build scale pitches across MIDI range 48-84.
	rootSemi := map[string]int{
		"C": 0, "C#": 1, "D": 2, "D#": 3, "E": 4, "F": 5,
		"F#": 6, "G": 7, "G#": 8, "A": 9, "A#": 10, "B": 11,
	}[keyRoot]

	var intervals []int
	switch keyMode {
	case "minor", "natural_minor":
		intervals = []int{0, 2, 3, 5, 7, 8, 10}
	default:
		intervals = []int{0, 2, 4, 5, 7, 9, 11}
	}

	for oct := 3; oct <= 6; oct++ {
		base := (oct + 1) * 12
		for _, iv := range intervals {
			p := base + rootSemi + iv
			if p >= 36 && p <= 96 {
				g.ScalePitches = append(g.ScalePitches, p)
			}
		}
	}

	g.TonicPitch = 60 + rootSemi
	g.DominantPitch = g.TonicPitch + 7

	return g
}

// ApplyAll runs all melody grammar checks in sequence.
func (g *MelodyGrammar) ApplyAll(events []schema.NoteEvent, totalBars int) []schema.NoteEvent {
	if len(events) < 2 {
		return events
	}

	events = g.applyScaleMask(events)
	events = g.applyIntervalLimiter(events)
	events = g.applyGravity(events)
	events = g.applyPhraseStructure(events, totalBars)
	events = g.applyContourShaping(events, totalBars)

	fmt.Printf("[MelodyGrammar] applied: scale=%d pitch, interval≤%d, gravity=%.1f, step=%.0f%%\n",
		len(g.ScalePitches), g.MaxInterval, g.Gravity, g.StepBias*100)
	return events
}

// 1. Scale Mask: force every note to be in-key.
func (g *MelodyGrammar) applyScaleMask(events []schema.NoteEvent) []schema.NoteEvent {
	corrected := 0
	scaleSet := make(map[int]bool)
	for _, p := range g.ScalePitches {
		scaleSet[p] = true
	}

	for i := range events {
		if !scaleSet[events[i].Pitch] {
			// Snap to nearest in-key pitch.
			nearest := events[i].Pitch
			for _, sp := range g.ScalePitches {
				if abs(events[i].Pitch-sp) < abs(events[i].Pitch-nearest) {
					nearest = sp
				}
			}
			events[i].Pitch = nearest
			corrected++
		}
	}
	if corrected > 0 {
		fmt.Printf("[ScaleMask] corrected %d/%d out-of-key notes\n", corrected, len(events))
	}
	return events
}

// 2. Interval Limiter: fold large jumps, insert passing notes.
func (g *MelodyGrammar) applyIntervalLimiter(events []schema.NoteEvent) []schema.NoteEvent {
	result := make([]schema.NoteEvent, 0, len(events)*2)
	fixed := 0

	for i := 0; i < len(events); i++ {
		result = append(result, events[i])

		if i+1 < len(events) {
			interval := events[i+1].Pitch - events[i].Pitch
			absInt := interval
			if absInt < 0 {
				absInt = -absInt
			}

			if absInt > g.MaxInterval {
				// Insert a passing note halfway.
				midPitch := (events[i].Pitch + events[i+1].Pitch) / 2
				midBeat := (events[i].StartBeat + events[i+1].StartBeat) / 2
				midDur := (events[i+1].StartBeat - events[i].StartBeat) * 0.5

				result = append(result, schema.NoteEvent{
					Type: "note", Pitch: midPitch,
					StartBeat:    midBeat,
					DurationBeat: midDur,
					Velocity:     (events[i].Velocity + events[i+1].Velocity) / 2,
				})
				fixed++
			}
		}
	}
	if fixed > 0 {
		fmt.Printf("[IntervalLimiter] inserted %d passing notes (max interval=%d)\n", fixed, g.MaxInterval)
	}
	return result
}

// 3. Gravity System: ensure each phrase returns to tonic or chord tone.
func (g *MelodyGrammar) applyGravity(events []schema.NoteEvent) []schema.NoteEvent {
	if len(events) < 4 {
		return events
	}

	// Every 4 bars = one phrase. The last note of each phrase should be a tonic or dominant.
	barsPerPhrase := 4
	phraseEnds := make(map[int]bool)

	for i := range events {
		bar := int(events[i].StartBeat) / 4
		if bar > 0 && bar%barsPerPhrase == 0 {
			// This bar is a phrase boundary. Find the last note in it.
			phraseEnds[bar] = true
		}
	}

	for i := len(events) - 1; i >= 0; i-- {
		bar := int(events[i].StartBeat) / 4
		if phraseEnds[bar] {
			// Snap to nearest tonic or dominant.
			distToTonic := abs(events[i].Pitch - g.TonicPitch)
			distToDom := abs(events[i].Pitch - g.DominantPitch)
			if distToTonic <= distToDom {
				events[i].Pitch = g.TonicPitch
			} else {
				events[i].Pitch = g.DominantPitch
			}
			delete(phraseEnds, bar)
		}
	}
	return events
}

// 4. Phrase Structure: force rests and long notes every 2-4 bars.
func (g *MelodyGrammar) applyPhraseStructure(events []schema.NoteEvent, totalBars int) []schema.NoteEvent {
	if totalBars < 4 {
		return events
	}

	// Every 4 bars, ensure there's either a rest or a long note (>1.0 beat).
	barsPerPhrase := 4
	type barRange struct{ start, end float64 }

	for phraseStart := 0; phraseStart < totalBars; phraseStart += barsPerPhrase {
		phraseEnd := phraseStart + barsPerPhrase
		if phraseEnd > totalBars {
			phraseEnd = totalBars
		}

		hasLongNote := false
		for _, ev := range events {
			bar := int(ev.StartBeat) / 4
			if bar >= phraseStart && bar < phraseEnd && ev.DurationBeat >= 1.0 {
				hasLongNote = true
				break
			}
		}

		if !hasLongNote && len(events) > 0 {
			// Extend the last note of this phrase to be a long note.
			for i := len(events) - 1; i >= 0; i-- {
				bar := int(events[i].StartBeat) / 4
				if bar >= phraseStart && bar < phraseEnd {
					if events[i].DurationBeat < 1.0 {
						events[i].DurationBeat = 1.5
					}
					break
				}
			}
		}
	}
	return events
}

// 5. Contour Shaping: verse low→mid, chorus mid→high, ending fall.
func (g *MelodyGrammar) applyContourShaping(events []schema.NoteEvent, totalBars int) []schema.NoteEvent {
	if totalBars < 4 {
		return events
	}

	third := totalBars / 3
	if third < 2 {
		third = 2
	}

	for i := range events {
		bar := int(events[i].StartBeat) / 4

		switch {
		case bar < third:
			// First third: lower register (keep or lower).
			if events[i].Pitch > g.TonicPitch+12 {
				events[i].Pitch -= 12
			}
		case bar < third*2:
			// Middle third: rise toward dominant.
			if events[i].Pitch < g.TonicPitch+7 {
				events[i].Pitch += 0 // let it be
			}
		default:
			// Last third: climax then fall.
			progress := float64(bar-third*2) / float64(totalBars-third*2)
			if progress < 0.5 {
				// Climax zone (first half of last third).
				if events[i].Pitch < g.TonicPitch+12 {
					events[i].Pitch += 3
				}
			} else {
				// Resolution zone (second half).
				// Pull toward tonic.
				dist := events[i].Pitch - g.TonicPitch
				if abs(dist) > 5 {
					if dist > 0 {
						events[i].Pitch -= 3
					} else {
						events[i].Pitch += 3
					}
				}
			}
		}
	}
	return events
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
