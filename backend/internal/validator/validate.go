// Package validator provides MIDI IR quality checks and auto-fixes.
// Ported concepts from Clef's music21 validate_abc.py + abc_lint.py + fix_measure_duration.
package validator

import (
	"fmt"
	"math"
	"strings"

	"github.com/ShowerBandV/text2midi/internal/schema"
)

// Report holds the result of a validation run.
type Report struct {
	Passed  bool     `json:"passed"`
	Score   float64  `json:"score"` // 0-100
	Errors  []string `json:"errors"`
	Warnings []string `json:"warnings"`
	Fixes   []string `json:"fixes,omitempty"` // what was auto-fixed
}

// Validate runs all checks on a MidiIR and returns a report.
// If autoFix is true, it attempts to fix measure duration issues.
func Validate(mid schema.MidiIR, totalBars int, autoFix bool) *Report {
	r := &Report{Passed: true, Score: 100.0}

	// 1. Track count.
	if len(mid.Tracks) == 0 {
		r.Errors = append(r.Errors, "no tracks in MIDI IR")
		r.Passed = false
		r.Score = 0
		return r
	}

	// 2. Per-track checks.
	allEmpty := true
	for _, t := range mid.Tracks {
		if !t.Enabled {
			continue
		}
		if len(t.Events) == 0 {
			r.Warnings = append(r.Warnings, fmt.Sprintf("track %q has no events", t.ID))
			continue
		}
		allEmpty = false

		// 2a. Pitch range.
		minP, maxP := 127, 0
		for _, ev := range t.Events {
			if ev.Type != "note" {
				continue
			}
			if ev.Pitch < minP {
				minP = ev.Pitch
			}
			if ev.Pitch > maxP {
				maxP = ev.Pitch
			}
			if ev.Pitch < 0 || ev.Pitch > 127 {
				r.Errors = append(r.Errors, fmt.Sprintf("track %q: note pitch %d out of [0-127]", t.ID, ev.Pitch))
				r.Passed = false
			}
			if ev.DurationBeat <= 0 {
				r.Errors = append(r.Errors, fmt.Sprintf("track %q: note at beat %.2f has duration %.2f ≤ 0", t.ID, ev.StartBeat, ev.DurationBeat))
				r.Passed = false
			}
			if ev.StartBeat < 0 {
				r.Errors = append(r.Errors, fmt.Sprintf("track %q: note has negative start beat %.2f", t.ID, ev.StartBeat))
				r.Passed = false
			}
		}
		pitchRange := maxP - minP
		if pitchRange < 3 && t.Channel != 9 { // drums can have narrow range
			r.Warnings = append(r.Warnings, fmt.Sprintf("track %q: very narrow pitch range (%d semitones)", t.ID, pitchRange))
		}
		if pitchRange > 60 {
			r.Warnings = append(r.Warnings, fmt.Sprintf("track %q: very wide pitch range (%d semitones)", t.ID, pitchRange))
		}

		// 2b. Note overlaps.
		overlaps := checkOverlaps(t.Events)
		if overlaps > 0 {
			r.Warnings = append(r.Warnings, fmt.Sprintf("track %q: %d overlapping notes detected", t.ID, overlaps))
		}
	}

	if allEmpty {
		r.Errors = append(r.Errors, "all tracks are empty")
		r.Passed = false
		r.Score = 10
		return r
	}

	// 3. Measure duration check.
	durErrors := checkMeasureDurations(mid.Tracks, totalBars, mid.Meta.TimeSignature)
	if len(durErrors) > 0 {
		if autoFix {
			fixes := FixMeasureDurations(mid.Tracks, totalBars, mid.Meta.TimeSignature)
			r.Fixes = append(r.Fixes, fixes...)
		} else {
			for _, e := range durErrors {
				r.Errors = append(r.Errors, e)
			}
			r.Passed = len(r.Errors) == 0
		}
	}

	// 4. Score calculation.
	penalty := len(r.Errors)*10 + len(r.Warnings)*2
	r.Score = math.Max(0, float64(100-penalty))

	return r
}

// checkOverlaps counts overlapping notes on a single track.
func checkOverlaps(events []schema.NoteEvent) int {
	count := 0
	for i := 0; i < len(events); i++ {
		if events[i].Type != "note" {
			continue
		}
		endA := events[i].StartBeat + events[i].DurationBeat
		for j := i + 1; j < len(events); j++ {
			if events[j].Type != "note" {
				continue
			}
			if events[j].StartBeat >= endA {
				break // events are sorted by start beat (assumption)
			}
			if events[j].StartBeat < endA && events[j].Pitch == events[i].Pitch {
				count++
			}
		}
	}
	return count
}

// checkMeasureDurations checks that each bar's total note duration matches the time signature.
// Drums (ch9) are excluded because drum events don't fill beat duration.
func checkMeasureDurations(tracks []schema.TrackIR, totalBars int, ts schema.TimeSignature) []string {
	var errors []string
	beatsPerBar := ts.Numerator
	if beatsPerBar <= 0 {
		beatsPerBar = 4
	}

	for _, t := range tracks {
		if t.Channel == 9 || !t.Enabled || len(t.Events) == 0 {
			continue
		}
		barDurations := make([]float64, totalBars)
		for _, ev := range t.Events {
			if ev.Type != "note" {
				continue
			}
			bar := int(ev.StartBeat) / beatsPerBar
			if bar >= totalBars {
				bar = totalBars - 1
			}
			barDurations[bar] += ev.DurationBeat
		}

		for bar, dur := range barDurations {
			if dur == 0 {
				continue // empty bar — OK
			}
			deviation := math.Abs(dur - float64(beatsPerBar))
			if deviation > 1.0 { // more than 1 beat off
				errors = append(errors, fmt.Sprintf(
					"track %q bar %d: measure duration %.1f, expected %d (off by %.1f beats)",
					t.ID, bar, dur, beatsPerBar, deviation))
			} else if deviation > 0.1 {
				// Small deviation — warning only.
				// (printed directly, not added to errors)
				fmt.Printf("  [Validate] track %q bar %d: slight timing drift (%.1f vs %d)\n",
					t.ID, bar, dur, beatsPerBar)
			}
		}
	}
	return errors
}

// FormatReport returns a human-readable summary of the validation report.
func FormatReport(r *Report) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Validation: %s (score: %.0f/100)\n",
		map[bool]string{true: "PASSED", false: "FAILED"}[r.Passed], r.Score))

	if len(r.Errors) > 0 {
		sb.WriteString(fmt.Sprintf("  Errors (%d):\n", len(r.Errors)))
		for _, e := range r.Errors {
			sb.WriteString(fmt.Sprintf("    - %s\n", e))
		}
	}
	if len(r.Warnings) > 0 {
		sb.WriteString(fmt.Sprintf("  Warnings (%d):\n", len(r.Warnings)))
		for _, w := range r.Warnings {
			sb.WriteString(fmt.Sprintf("    - %s\n", w))
		}
	}
	if len(r.Fixes) > 0 {
		sb.WriteString(fmt.Sprintf("  Auto-fixes (%d):\n", len(r.Fixes)))
		for _, f := range r.Fixes {
			sb.WriteString(fmt.Sprintf("    - %s\n", f))
		}
	}
	return sb.String()
}
