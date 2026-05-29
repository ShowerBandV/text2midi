// Package plan — Self-check (post-generation validation).
// Validates generated MIDI against the composition plan.
package plan

import (
	"fmt"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// ValidateResult holds the validation outcome.
type ValidateResult struct {
	Passed  bool
	Issues  []string
	Score   float64 // 0-1 quality score
}

// Validate checks generated events against the composition plan.
func Validate(eventsByTrack map[string][]schema.NoteEvent, p *Plan) *ValidateResult {
	result := &ValidateResult{Passed: true, Score: 1.0}

	// 1. Bar count check.
	leadEvents := eventsByTrack["lead"]
	if len(leadEvents) == 0 {
		result.Issues = append(result.Issues, "no lead track found")
		result.Passed = false
		result.Score = 0
		return result
	}

	maxBeat := 0.0
	for _, ev := range leadEvents {
		if ev.StartBeat > maxBeat {
			maxBeat = ev.StartBeat
		}
	}
	actualBars := int(maxBeat/4.0) + 1
	if actualBars != p.TotalBars {
		result.Issues = append(result.Issues,
			fmt.Sprintf("bar count mismatch: plan=%d, actual=%d", p.TotalBars, actualBars))
		result.Score -= 0.2
	}

	// 2. Pitch range check.
	minPitch, maxPitch := 127, 0
	for _, ev := range leadEvents {
		if ev.Pitch < minPitch {
			minPitch = ev.Pitch
		}
		if ev.Pitch > maxPitch {
			maxPitch = ev.Pitch
		}
	}
	pitchRange := maxPitch - minPitch
	if pitchRange < 5 {
		result.Issues = append(result.Issues,
			fmt.Sprintf("pitch range too narrow: %d semitones (need >5)", pitchRange))
		result.Score -= 0.15
	}
	if pitchRange > 36 {
		result.Issues = append(result.Issues,
			fmt.Sprintf("pitch range too wide: %d semitones (need <36)", pitchRange))
		result.Score -= 0.15
	}

	// 3. Velocity variation check.
	velSeen := make(map[int]bool)
	for _, ev := range leadEvents {
		velSeen[ev.Velocity] = true
	}
	if len(velSeen) < 3 {
		result.Issues = append(result.Issues,
			fmt.Sprintf("velocity too uniform: only %d distinct values", len(velSeen)))
		result.Score -= 0.1
	}

	// 4. Silence/rest check.
	// At least one bar should have < 4 notes (rests exist).
	notesPerBar := make(map[int]int)
	for _, ev := range leadEvents {
		bar := int(ev.StartBeat) / 4
		notesPerBar[bar]++
	}
	hasRest := false
	for _, count := range notesPerBar {
		if count < 4 {
			hasRest = true
			break
		}
	}
	if !hasRest {
		result.Issues = append(result.Issues, "no rests found — every bar is full")
		result.Score -= 0.1
	}

	// 5. Duration variety check.
	durSeen := make(map[float64]int)
	for _, ev := range leadEvents {
		durSeen[ev.DurationBeat]++
	}
	if len(durSeen) < 2 {
		result.Issues = append(result.Issues, "duration too uniform — no variety")
		result.Score -= 0.1
	}

	if result.Score < 0.5 {
		result.Passed = false
	}

	fmt.Printf("[Validate] passed=%t score=%.2f issues=%d\n", result.Passed, result.Score, len(result.Issues))
	for _, issue := range result.Issues {
		fmt.Printf("  - %s\n", issue)
	}
	return result
}
